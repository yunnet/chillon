package main

import (
	"github.com/yunnet/chillon/server"
	"log"
	"os"
)

func main() {
	logfile, err := os.Create("chillon.log")
	if err != nil{
		log.Fatal("fail to create chillon.log file.")
	}
	logger := log.New(logfile, "", log.Llongfile)


	upload := "./upload"
	listenPort := 2121

	_, err = os.Stat(upload)
	if os.IsNotExist(err){
		os.MkdirAll(upload, os.ModePerm)
	}

	perm := server.NewSimplePerm("root", "root")
	factory := &server.FileDriverFactory{
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
		Port:           listenPort,
		TLS:            false,
		CertFile:       "",
		KeyFile:        "",
		ExplicitFTPS:   false,
		WelcomeMessage: "welcome ftp server",
		Logger:         nil,
	}
	ftpserver := server.NewServer(opt)

	logger.Println("FTP Server start...", 2121)

	if err := ftpserver.ListenAndServe(); err != nil{
		logger.Fatal("Error starting server: ", err)
	}
}
