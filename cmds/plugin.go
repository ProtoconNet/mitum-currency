package cmds

type PluginCommand struct {
	RegisterModel RegisterContractCommand `cmd:"" name:"register-contract" help:"register contract"`
}
