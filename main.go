package main

import (
	"github.com/lunny/log"
	"github.com/yunnet/chillon/filedriver"
	"github.com/yunnet/chillon/server"
	"os"
)

func main() {
	upload := "./upload"
	_, err := os.Stat(upload)
	if os.IsNotExist(err){
		os.MkdirAll(upload, os.ModePerm)
	}

	perm := server.NewSimplePerm("root", "root")
	factory := &filedriver.FileDriverFactory{
		RootPath: upload,
		Perm:     perm,
	}

	auth := &server.SimpleAuth{Name:"admin", Password:"123456"}

	opt := &server.ServerOpts{
		Factory:        factory,
		Auth:           auth,
		Name:           "FtpServer",
		Hostname:       "",
		PublicIp:       "",
		PassivePorts:   "",
		Port:           2121,
		TLS:            false,
		CertFile:       "",
		KeyFile:        "",
		ExplicitFTPS:   false,
		WelcomeMessage: "welcome ftp server",
		Logger:         nil,
	}
	ftpserver := server.NewServer(opt)

	log.Info("FTP Server start...", 2121)

	if err := ftpserver.ListenAndServe(); err != nil{
		log.Fatal("Error starting server: ", err)
	}
}
