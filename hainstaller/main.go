// Currently used for local testing purposes

package main

func testDownload() {
	hai := NewHostAgentInstaller("placeholder", "/home/vmware/test123")
	hai.Download("1.2.1")
}

func main() {
	hai := NewHostAgentInstaller("asd", "/home/vmware/test123")
	var osName = hai.GetHostOS()
	println(osName)
	//hai.Uninstall()

	//testDownload()
}
