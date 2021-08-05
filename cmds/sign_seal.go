package cmds

import (
	"golang.org/x/xerrors"

	mitumcmds "github.com/spikeekips/mitum/launch/cmds"
	"github.com/spikeekips/mitum/util"
)

type SignSealCommand struct {
	*BaseCommand
	Privatekey PrivatekeyFlag          `arg:"" name:"privatekey" help:"sender's privatekey" required:"true"`
	NetworkID  mitumcmds.NetworkIDFlag `name:"network-id" help:"network-id" required:"true"`
	Pretty     bool                    `name:"pretty" help:"pretty format"`
	Seal       FileLoad                `help:"seal" optional:""`
}

func NewSignSealCommand() SignSealCommand {
	return SignSealCommand{
		BaseCommand: NewBaseCommand("sign-seal"),
	}
}

func (cmd *SignSealCommand) Run(version util.Version) error {
	if err := cmd.Initialize(cmd, version); err != nil {
		return xerrors.Errorf("failed to initialize command: %w", err)
	}

	sl, err := loadSeal(cmd.Seal.Bytes(), cmd.NetworkID.NetworkID())
	if err != nil {
		return err
	}

	cmd.Log().Debug().Stringer("seal", sl.Hash()).Msg("seal loaded")

	if sl.Signer().Equal(cmd.Privatekey.Publickey()) {
		cmd.Log().Debug().Msg("already signed")
	} else {
		s, err := signSeal(sl, cmd.Privatekey, cmd.NetworkID.NetworkID())
		if err != nil {
			return err
		}
		cmd.Log().Debug().Msg("seal signed")

		sl = s
	}

	cmd.pretty(cmd.Pretty, sl)

	return nil
}
