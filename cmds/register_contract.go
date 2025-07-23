package cmds

import (
	"context"
	"github.com/ProtoconNet/mitum-currency/v3/operation/plugin"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/util"
	"github.com/pkg/errors"
)

type RegisterContractCommand struct {
	BaseCommand
	OperationFlags
	Sender       AddressFlag    `arg:"" name:"sender" help:"sender address" required:"true"`
	Contract     AddressFlag    `arg:"" name:"contract" help:"contract account to register policy" required:"true"`
	ContractCode string         `arg:"" name:"contract-code" help:"contract code" type:"filepath"`
	Currency     CurrencyIDFlag `arg:"" name:"currency" help:"currency id" required:"true"`
	sender       base.Address
	contract     base.Address
}

func (cmd *RegisterContractCommand) Run(pctx context.Context) error {
	if _, err := cmd.prepare(pctx); err != nil {
		return err
	}

	if err := cmd.parseFlags(); err != nil {
		return err
	}

	op, err := cmd.createOperation()
	if err != nil {
		return err
	}

	PrettyPrint(cmd.Out, op)

	return nil
}

func (cmd *RegisterContractCommand) parseFlags() error {
	if err := cmd.OperationFlags.IsValid(nil); err != nil {
		return err
	}

	if a, err := cmd.Sender.Encode(cmd.Encoders.JSON()); err != nil {
		return errors.Wrapf(err, "invalid sender format; %q", cmd.Sender)
	} else {
		cmd.sender = a
	}

	if a, err := cmd.Contract.Encode(cmd.Encoders.JSON()); err != nil {
		return errors.Wrapf(err, "invalid contract format; %q", cmd.Contract)
	} else {
		cmd.contract = a
	}

	return nil
}

func (cmd *RegisterContractCommand) createOperation() (base.Operation, error) {
	e := util.StringError("failed to create register-model operation")

	//wasmBytes, err := os.ReadFile(cmd.ContractCode)
	//if err != nil {
	//	return nil, e.Wrap(err)
	//}
	//encodedCodeString := base64.StdEncoding.EncodeToString(wasmBytes)
	fact := plugin.NewRegisterModelFact([]byte(cmd.Token), cmd.sender, cmd.contract, cmd.ContractCode, cmd.Currency.CID)

	op, err := plugin.NewRegisterModel(fact)
	if err != nil {
		return nil, e.Wrap(err)
	}
	err = op.Sign(cmd.Privatekey, cmd.NetworkID.NetworkID())
	if err != nil {
		return nil, e.Wrap(err)
	}

	return op, nil
}
