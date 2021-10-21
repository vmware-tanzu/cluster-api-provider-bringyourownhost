package algo

type OutputBuilder interface {
	StdOut(string)
	StdErr(string)
	Cmd(string)
	Desc(string)
	Msg(string)
}
