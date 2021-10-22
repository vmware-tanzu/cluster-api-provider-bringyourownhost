package algo

type OutputBuilder interface {
	Out(string)
	Err(string)
	Cmd(string)
	Desc(string)
	Msg(string)
}
