package cmds

import (
	"github.com/ProtoconNet/mitum-currency/v3/common"
	"github.com/ProtoconNet/mitum-currency/v3/digest"
	digestisaac "github.com/ProtoconNet/mitum-currency/v3/digest/isaac"
	"github.com/ProtoconNet/mitum-currency/v3/operation/currency"
	"github.com/ProtoconNet/mitum-currency/v3/operation/did-registry"
	"github.com/ProtoconNet/mitum-currency/v3/operation/extension"
	"github.com/ProtoconNet/mitum-currency/v3/operation/extras"
	isaacoperation "github.com/ProtoconNet/mitum-currency/v3/operation/isaac"
	ccstate "github.com/ProtoconNet/mitum-currency/v3/state/currency"
	dstate "github.com/ProtoconNet/mitum-currency/v3/state/did-registry"
	cestate "github.com/ProtoconNet/mitum-currency/v3/state/extension"
	"github.com/ProtoconNet/mitum-currency/v3/types"
	"github.com/ProtoconNet/mitum2/base"
	"github.com/ProtoconNet/mitum2/launch"
	"github.com/ProtoconNet/mitum2/util/encoder"
	"github.com/ProtoconNet/mitum2/util/hint"
	"github.com/pkg/errors"
)

var Hinters []encoder.DecodeDetail
var SupportedProposalOperationFactHinters []encoder.DecodeDetail

var AddedHinters = []encoder.DecodeDetail{
	// revive:disable-next-line:line-length-limit
	{Hint: common.BaseStateHint, Instance: common.BaseState{}},
	{Hint: common.NodeHint, Instance: common.BaseNode{}},

	{Hint: types.AccountHint, Instance: types.Account{}},
	{Hint: types.AccountKeyHint, Instance: types.BaseAccountKey{}},
	{Hint: types.AccountKeysHint, Instance: types.BaseAccountKeys{}},
	{Hint: types.NilAccountKeysHint, Instance: types.NilAccountKeys{}},
	{Hint: types.AddressHint, Instance: types.Address{}},
	{Hint: types.StringAddressHint, Instance: types.StringAddress{}},
	{Hint: types.AmountHint, Instance: types.Amount{}},
	{Hint: types.ContractAccountKeysHint, Instance: types.ContractAccountKeys{}},
	{Hint: types.ContractAccountStatusHint, Instance: types.ContractAccountStatus{}},
	{Hint: types.CurrencyDesignHint, Instance: types.CurrencyDesign{}},
	{Hint: types.CurrencyPolicyHint, Instance: types.CurrencyPolicy{}},
	{Hint: types.FixedFeeerHint, Instance: types.FixedFeeer{}},
	{Hint: types.MEPrivatekeyHint, Instance: types.MEPrivatekey{}},
	{Hint: types.MEPublickeyHint, Instance: types.MEPublickey{}},
	{Hint: types.NilFeeerHint, Instance: types.NilFeeer{}},
	{Hint: types.RatioFeeerHint, Instance: types.RatioFeeer{}},

	{Hint: currency.CreateAccountHint, Instance: currency.CreateAccount{}},
	{Hint: currency.CreateAccountItemMultiAmountsHint, Instance: currency.CreateAccountItemMultiAmounts{}},
	{Hint: currency.CreateAccountItemSingleAmountHint, Instance: currency.CreateAccountItemSingleAmount{}},
	{Hint: currency.UpdateCurrencyHint, Instance: currency.UpdateCurrency{}},
	{Hint: currency.RegisterCurrencyHint, Instance: currency.RegisterCurrency{}},
	{Hint: currency.RegisterGenesisCurrencyHint, Instance: currency.RegisterGenesisCurrency{}},
	{Hint: currency.RegisterGenesisCurrencyFactHint, Instance: currency.RegisterGenesisCurrencyFact{}},
	{Hint: currency.UpdateKeyHint, Instance: currency.UpdateKey{}},
	{Hint: currency.MintHint, Instance: currency.Mint{}},
	{Hint: currency.TransferHint, Instance: currency.Transfer{}},
	{Hint: currency.TransferItemMultiAmountsHint, Instance: currency.TransferItemMultiAmounts{}},
	{Hint: currency.TransferItemSingleAmountHint, Instance: currency.TransferItemSingleAmount{}},

	{Hint: extension.CreateContractAccountHint, Instance: extension.CreateContractAccount{}},
	{Hint: extension.CreateContractAccountItemMultiAmountsHint, Instance: extension.CreateContractAccountItemMultiAmounts{}},
	{Hint: extension.CreateContractAccountItemSingleAmountHint, Instance: extension.CreateContractAccountItemSingleAmount{}},
	{Hint: extension.UpdateHandlerHint, Instance: extension.UpdateHandler{}},
	{Hint: extension.UpdateRecipientHint, Instance: extension.UpdateRecipient{}},
	{Hint: extension.WithdrawHint, Instance: extension.Withdraw{}},
	{Hint: extension.WithdrawItemMultiAmountsHint, Instance: extension.WithdrawItemMultiAmounts{}},
	{Hint: extension.WithdrawItemSingleAmountHint, Instance: extension.WithdrawItemSingleAmount{}},

	{Hint: extras.BaseAuthenticationHint, Instance: extras.BaseAuthentication{}},
	{Hint: extras.BaseSettlementHint, Instance: extras.BaseSettlement{}},
	{Hint: extras.BaseProxyPayerHint, Instance: extras.BaseProxyPayer{}},

	{Hint: isaacoperation.GenesisNetworkPolicyHint, Instance: isaacoperation.GenesisNetworkPolicy{}},
	{Hint: isaacoperation.FixedSuffrageCandidateLimiterRuleHint, Instance: isaacoperation.FixedSuffrageCandidateLimiterRule{}},
	{Hint: isaacoperation.MajoritySuffrageCandidateLimiterRuleHint, Instance: isaacoperation.MajoritySuffrageCandidateLimiterRule{}},
	{Hint: types.NetworkPolicyHint, Instance: types.NetworkPolicy{}},
	{Hint: types.NetworkPolicyStateValueHint, Instance: types.NetworkPolicyStateValue{}},
	{Hint: isaacoperation.SuffrageCandidateHint, Instance: isaacoperation.SuffrageCandidate{}},
	{Hint: isaacoperation.SuffrageDisjoinHint, Instance: isaacoperation.SuffrageDisjoin{}},
	{Hint: isaacoperation.SuffrageGenesisJoinHint, Instance: isaacoperation.SuffrageGenesisJoin{}},
	{Hint: isaacoperation.SuffrageJoinHint, Instance: isaacoperation.SuffrageJoin{}},
	{Hint: isaacoperation.NetworkPolicyHint, Instance: isaacoperation.NetworkPolicy{}},

	{Hint: ccstate.AccountStateValueHint, Instance: ccstate.AccountStateValue{}},
	{Hint: ccstate.BalanceStateValueHint, Instance: ccstate.BalanceStateValue{}},
	{Hint: ccstate.DesignStateValueHint, Instance: ccstate.DesignStateValue{}},

	{Hint: cestate.ContractAccountStateValueHint, Instance: cestate.ContractAccountStateValue{}},

	{Hint: digest.AccountValueHint, Instance: digest.AccountValue{}},
	{Hint: digest.OperationValueHint, Instance: digest.OperationValue{}},
	{Hint: digestisaac.ManifestHint, Instance: digestisaac.Manifest{}},

	{Hint: types.DesignHint, Instance: types.Design{}},
	{Hint: types.DataHint, Instance: types.Data{}},
	{Hint: types.DIDResourceHint, Instance: types.DIDResource{}},
	{Hint: types.DIDDocumentHint, Instance: types.DIDDocument{}},
	{Hint: types.AsymmetricKeyAuthenticationHint, Instance: types.AsymmetricKeyAuthentication{}},
	{Hint: types.SocialLogInAuthenticationHint, Instance: types.SocialLogInAuthentication{}},

	{Hint: did_registry.CreateDIDHint, Instance: did_registry.CreateDID{}},
	{Hint: did_registry.UpdateDIDDocumentHint, Instance: did_registry.UpdateDIDDocument{}},
	{Hint: did_registry.RegisterModelHint, Instance: did_registry.RegisterModel{}},
	{Hint: dstate.DataStateValueHint, Instance: dstate.DataStateValue{}},
	{Hint: dstate.DesignStateValueHint, Instance: dstate.DesignStateValue{}},
	{Hint: dstate.DocumentStateValueHint, Instance: dstate.DocumentStateValue{}},
}

var AddedSupportedHinters = []encoder.DecodeDetail{
	{Hint: currency.CreateAccountFactHint, Instance: currency.CreateAccountFact{}},
	{Hint: currency.UpdateCurrencyFactHint, Instance: currency.UpdateCurrencyFact{}},
	{Hint: currency.RegisterCurrencyFactHint, Instance: currency.RegisterCurrencyFact{}},
	{Hint: currency.UpdateKeyFactHint, Instance: currency.UpdateKeyFact{}},
	{Hint: currency.MintFactHint, Instance: currency.MintFact{}},
	{Hint: currency.TransferFactHint, Instance: currency.TransferFact{}},

	{Hint: extension.CreateContractAccountFactHint, Instance: extension.CreateContractAccountFact{}},
	{Hint: extension.UpdateHandlerFactHint, Instance: extension.UpdateHandlerFact{}},
	{Hint: extension.UpdateRecipientFactHint, Instance: extension.UpdateRecipientFact{}},
	{Hint: extension.WithdrawFactHint, Instance: extension.WithdrawFact{}},

	{Hint: isaacoperation.GenesisNetworkPolicyFactHint, Instance: isaacoperation.GenesisNetworkPolicyFact{}},
	{Hint: isaacoperation.SuffrageCandidateFactHint, Instance: isaacoperation.SuffrageCandidateFact{}},
	{Hint: isaacoperation.SuffrageDisjoinFactHint, Instance: isaacoperation.SuffrageDisjoinFact{}},
	{Hint: isaacoperation.SuffrageGenesisJoinFactHint, Instance: isaacoperation.SuffrageGenesisJoinFact{}},
	{Hint: isaacoperation.SuffrageJoinFactHint, Instance: isaacoperation.SuffrageJoinFact{}},
	{Hint: isaacoperation.NetworkPolicyFactHint, Instance: isaacoperation.NetworkPolicyFact{}},

	{Hint: did_registry.CreateDIDFactHint, Instance: did_registry.CreateDIDFact{}},
	{Hint: did_registry.UpdateDIDDocumentFactHint, Instance: did_registry.UpdateDIDDocumentFact{}},
	{Hint: did_registry.RegisterModelFactHint, Instance: did_registry.RegisterModelFact{}},
}

func init() {
	hinters := ExcludeHint(base.StringAddressHint, launch.Hinters)
	Hinters = make([]encoder.DecodeDetail, len(hinters)+len(AddedHinters))
	copy(Hinters, hinters)
	copy(Hinters[len(hinters):], AddedHinters)

	SupportedProposalOperationFactHinters = make(
		[]encoder.DecodeDetail,
		len(launch.SupportedProposalOperationFactHinters)+len(AddedSupportedHinters),
	)
	copy(SupportedProposalOperationFactHinters, launch.SupportedProposalOperationFactHinters)
	copy(SupportedProposalOperationFactHinters[len(launch.SupportedProposalOperationFactHinters):],
		AddedSupportedHinters,
	)
}

func LoadHinters(encs *encoder.Encoders) error {
	for i := range Hinters {
		if err := encs.AddDetail(Hinters[i]); err != nil {
			return errors.Wrap(err, "add hinter to encoder")
		}
	}

	for i := range SupportedProposalOperationFactHinters {
		if err := encs.AddDetail(SupportedProposalOperationFactHinters[i]); err != nil {
			return errors.Wrap(err, "add supported proposal operation fact hinter to encoder")
		}
	}

	return nil
}

func ExcludeHint(hint hint.Hint, launchHinters []encoder.DecodeDetail) []encoder.DecodeDetail {
	var hinters []encoder.DecodeDetail
	for _, v := range launchHinters {
		if !v.Hint.Equal(hint) {
			hinters = append(hinters, v)
		}
	}
	return hinters
}
