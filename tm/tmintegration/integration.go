package tmintegration

import (
	"context"
	"crypto/sha256"
	"fmt"
	"strings"
	"testing"
	"time"

	"github.com/rollchains/gordian/internal/gtest"
	"github.com/rollchains/gordian/tm/tmapp"
	"github.com/rollchains/gordian/tm/tmconsensus"
	"github.com/rollchains/gordian/tm/tmconsensus/tmconsensustest"
	"github.com/rollchains/gordian/tm/tmengine"
	"github.com/rollchains/gordian/tm/tmp2p"
	"github.com/stretchr/testify/require"
)

func RunIntegrationTest(t *testing.T, nf NewFactoryFunc) {
	t.Run("basic flow with identity app", func(t *testing.T) {
		t.Parallel()

		ctx, cancel := context.WithCancel(context.Background())
		defer cancel()

		log := gtest.NewLogger(t)
		f := nf(&Env{
			RootLogger: log,

			tb: t,
		})

		n, err := f.NewNetwork(ctx, log)
		require.NoError(t, err)
		defer n.Wait()
		defer cancel()

		fx := tmconsensustest.NewStandardFixture(2)
		genesis := fx.DefaultGenesis()

		// Make just the connections first, so we can stabilize the network,
		// before we begin instantiating the engines.
		conns := make([]tmp2p.Connection, len(fx.PrivVals))
		for i := range fx.PrivVals {
			conn, err := n.Connect(ctx)
			require.NoError(t, err)
			conns[i] = conn
		}

		require.NoError(t, n.Stabilize(ctx))

		apps := make([]*identityApp, len(fx.PrivVals))

		for i, v := range fx.PrivVals {
			hashScheme, err := f.HashScheme(ctx, i)
			require.NoError(t, err)

			sigScheme, err := f.SignatureScheme(ctx, i)
			require.NoError(t, err)

			cmspScheme, err := f.CommonMessageSignatureProofScheme(ctx, i)
			require.NoError(t, err)

			as, err := f.NewActionStore(ctx, i)
			require.NoError(t, err)

			bs, err := f.NewBlockStore(ctx, i)
			require.NoError(t, err)

			fs, err := f.NewFinalizationStore(ctx, i)
			require.NoError(t, err)

			ms, err := f.NewMirrorStore(ctx, i)
			require.NoError(t, err)

			rs, err := f.NewRoundStore(ctx, i)
			require.NoError(t, err)

			vs, err := f.NewValidatorStore(ctx, i, hashScheme)
			require.NoError(t, err)

			gStrat, err := f.NewGossipStrategy(ctx, i, conns[i])
			require.NoError(t, err)

			cStrat := &identityConsensusStrategy{
				Log:    log.With("sys", "consensusstrategy", "idx", i),
				PubKey: v.CVal.PubKey,
			}

			blockFinCh := make(chan tmapp.FinalizeBlockRequest)
			initChainCh := make(chan tmapp.InitChainRequest)

			app := newIdentityApp(
				ctx, log.With("sys", "app", "idx", i), i,
				initChainCh, blockFinCh,
			)
			t.Cleanup(app.Wait)
			t.Cleanup(cancel)

			apps[i] = app

			e, err := tmengine.New(
				ctx,
				log.With("sys", "engine", "idx", i),
				tmengine.WithActionStore(as),
				tmengine.WithBlockStore(bs),
				tmengine.WithFinalizationStore(fs),
				tmengine.WithMirrorStore(ms),
				tmengine.WithRoundStore(rs),
				tmengine.WithValidatorStore(vs),

				tmengine.WithHashScheme(hashScheme),
				tmengine.WithSignatureScheme(sigScheme),
				tmengine.WithCommonMessageSignatureProofScheme(cmspScheme),

				tmengine.WithGossipStrategy(gStrat),
				tmengine.WithConsensusStrategy(cStrat),

				tmengine.WithGenesis(&tmconsensus.ExternalGenesis{
					ChainID:           genesis.ChainID,
					InitialHeight:     genesis.InitialHeight,
					InitialAppState:   strings.NewReader(""), // No initial app state for identity app.
					GenesisValidators: fx.Vals(),
				}),

				// TODO: this might need scaled up to run on a slower machine.
				// Plus we really don't want to trigger any timeouts during these tests anyway.
				tmengine.WithTimeoutStrategy(ctx, tmengine.LinearTimeoutStrategy{
					ProposalBase: 250 * time.Millisecond,

					PrevoteDelayBase:   100 * time.Millisecond,
					PrecommitDelayBase: 100 * time.Millisecond,

					CommitWaitBase: 15 * time.Millisecond,
				}),

				tmengine.WithBlockFinalizationChannel(blockFinCh),
				tmengine.WithInitChainChannel(initChainCh),

				tmengine.WithSigner(v.Signer),
			)
			require.NoError(t, err)
			t.Cleanup(e.Wait)
			t.Cleanup(cancel)

			conns[i].SetConsensusHandler(tmconsensus.AcceptAllValidFeedbackMapper{
				Handler: e,
			})
		}

		for i := uint64(1); i < 6; i++ {
			t.Logf("Beginning finalization sync for height %d", i)
			for appIdx := 0; appIdx < len(apps); appIdx++ {
				finResp := gtest.ReceiveOrTimeout(t, apps[appIdx].FinalizeResponses, gtest.ScaleMs(1200))
				require.Equal(t, i, finResp.Height)

				round := finResp.Round

				expData := fmt.Sprintf("Height: %d; Round: %d", finResp.Height, round)
				expDataHash := sha256.Sum256([]byte(expData))
				require.Equal(t, expDataHash[:], finResp.AppStateHash)
			}
		}
	})
}