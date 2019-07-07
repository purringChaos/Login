package main

/*
#include <stdio.h>
#include <inttypes.h>
#include <stdlib.h>
#include <unistd.h>
#include <fcntl.h>
#include <sys/stat.h>
#include <sys/types.h>
#include <grp.h>

int openTTY() {
	return  open(getenv("TTY"), O_RDWR);
}
int setTTYOwner(int ttyfd, int uid, int gid) {
	return fchown(ttyfd, uid, gid) || fchmod(ttyfd, 0600);
}

void runShell() {
  execl("/bin/bash", "/bin/bash", (char *)0);
}

*/
import "C"

import (
	"errors"
	"fmt"
	"github.com/AlecAivazis/survey"
	"github.com/common-nighthawk/go-figure"
	lol "github.com/kris-nova/lolgopher"
	"github.com/msteinert/pam"
	"github.com/tudurom/ttyname"
	"log"
	"os"
	"os/exec"
	"os/user"
	"strconv"
	"strings"
)

var usernamesList []string

func usernamePrompt() (string, error) {
	username := ""
	prompt := &survey.Select{
		Message: "Username:",
		Options: usernamesList,
	}

	err := survey.AskOne(prompt, &username)
	return username, err
}
func remove(slice []int, s int) []int {
	return append(slice[:s], slice[s+1:]...)
}
func main() {
	usernamesList = []string{"kitteh", "root", "efwnfuwfu"}
	for x, y := range usernamesList {
		_, err := user.Lookup(y)
		if err != nil {
			usernamesList = append(usernamesList[:x], usernamesList[x+1:]...)
		}
	}

	w := lol.NewLolWriter()

	figure.Write(w, figure.NewFigure("Kitteh's Laptop", "", true))
	tty, _ := ttyname.TTY()
	ttyID := strings.ReplaceAll(tty, "/dev/tty", "")

	C.setenv(C.CString("XDG_SESSION_TYPE"), C.CString("tty"), C.int(1))
	C.setenv(C.CString("XDG_SEAT"), C.CString("seat0"), C.int(1))
	C.setenv(C.CString("XDG_VTNR"), C.CString(ttyID), C.int(1))

	t, err := pam.StartFunc("login", "", func(s pam.Style, msg string) (string, error) {
		switch s {
		case pam.PromptEchoOff:
			resp := ""
			prompt := &survey.Password{
				Message: strings.TrimSpace(strings.Title(msg)),
			}
			err := survey.AskOne(prompt, &resp)
			return resp, err

		case pam.PromptEchoOn:
			if msg == "login:" {
				return usernamePrompt()
			} else {
				resp := ""
				prompt := &survey.Input{
					Message: strings.Title(msg),
				}
				err := survey.AskOne(prompt, &resp)
				return resp, err
			}

		case pam.ErrorMsg:
			log.Print(msg)
			return "", nil
		case pam.TextInfo:
			fmt.Println(msg)
			return "", nil
		}
		return "", errors.New("Unrecognized message style")
	})
	if err != nil {
		log.Fatalf("Start: %s", err.Error())
	}
	err = t.Authenticate(0)
	if err != nil {
		log.Fatalf("Authenticate: %s", err.Error())
	}
	err = t.OpenSession(0)
	if err != nil {
		log.Fatalf("OpenSession: %s", err.Error())
	}
	username, _ := t.GetItem(pam.User)

	shellout, err := exec.Command("getent", "passwd", username).Output()
	if err != nil {
		log.Fatalf("GetShell: %s", err.Error())
	}

	ent := strings.Split(strings.TrimSuffix(string(shellout), "\n"), ":")
	shell := ent[6]
	home := ent[5]

	pamEnv, _ := t.GetEnvList()

	currentEnv := os.Environ()
	for _, b := range currentEnv {
		result := strings.SplitN(b, "=", 2)
		C.setenv(C.CString(result[0]), C.CString(result[1]), 1)
	}
	for x, b := range pamEnv {
		C.setenv(C.CString(x), C.CString(b), 1)
	}

	userC, err := user.Lookup(username)

	C.setenv(C.CString("HOME"), C.CString(home), 1)
	C.setenv(C.CString("USER"), C.CString(username), 1)
	C.setenv(C.CString("SHELL"), C.CString(shell), 1)
	C.setenv(C.CString("LOGNAME"), C.CString(username), 1)
	C.setenv(C.CString("KITTEHLOGIN"), C.CString("true"), 1)
	C.setenv(C.CString("TTY"), C.CString(tty), 1)

	//userC, err := user.Lookup(username)
	gid, _ := strconv.Atoi(userC.Gid)
	uid, _ := strconv.Atoi(userC.Uid)

	ttyfd := C.openTTY()
	C.setTTYOwner(C.int(ttyfd), C.int(uid), C.int(gid))
	C.initgroups(C.CString(username), C.uint(gid))
	C.setgid(C.uint(gid))
	C.setuid(C.uint(uid))

	C.chdir(C.getenv(C.CString("HOME")))
	C.runShell()
}
