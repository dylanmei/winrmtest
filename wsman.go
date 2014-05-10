package winrmtest

import (
	"bytes"
	"encoding/base64"
	"fmt"
	"net/http"
	"strconv"
	"strings"

	"github.com/masterzen/winrm/soap"
	"github.com/masterzen/xmlpath"
)

type wsman struct {
	commands     []command
	identitySeed int
}

type command struct {
	id      string
	text    string
	handler CommandFunc
}

func (w *wsman) HandleCommand(cmd string, f CommandFunc) string {
	w.commands = append(w.commands,
		command{
			id:      "456",
			text:    cmd,
			handler: f,
		})

	return "456"
}

func (w *wsman) CommandByText(cmd string) *command {
	for _, c := range w.commands {
		if c.text == cmd {
			return &c
		}
	}
	return nil
}

func (w *wsman) CommandById(id string) *command {
	for _, c := range w.commands {
		if c.id == id {
			return &c
		}
	}
	return nil
}

func (w *wsman) ServeHTTP(rw http.ResponseWriter, r *http.Request) {
	rw.Header().Add("Content-Type", "application/soap+xml")

	defer r.Body.Close()
	env, err := xmlpath.Parse(r.Body)

	if err != nil {
		return
	}

	action := readAction(env)
	if strings.HasSuffix(action, "transfer/Create") {
		// create a new shell

		rw.Write([]byte(`
			<env:Envelope xmlns:env="http://www.w3.org/2003/05/soap-envelope" xmlns:rsp="http://schemas.microsoft.com/wbem/wsman/1/windows/shell">
				<rsp:ShellId>123</rsp:ShellId>
			</env:Envelope>`))

	} else if strings.HasSuffix(action, "shell/Command") {
		// execute on behalf of the client
		text := readCommand(env)
		cmd := w.CommandByText(text)

		if cmd == nil {
			fmt.Printf("I don't know this command: Command=%s\n", text)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		rw.Write([]byte(fmt.Sprintf(`
			<env:Envelope xmlns:env="http://www.w3.org/2003/05/soap-envelope" xmlns:rsp="http://schemas.microsoft.com/wbem/wsman/1/windows/shell">
				<rsp:CommandId>%s</rsp:CommandId>
			</env:Envelope>`, cmd.id)))

	} else if strings.HasSuffix(action, "shell/Receive") {
		// client ready to receive the results

		id := readCommandIdFromDesiredStream(env)
		cmd := w.CommandById(id)

		if cmd == nil {
			fmt.Printf("I don't know this command: CommandId=%s\n", id)
			rw.WriteHeader(http.StatusInternalServerError)
			return
		}

		stdout := bytes.NewBuffer(make([]byte, 0))
		stderr := bytes.NewBuffer(make([]byte, 0))
		result := cmd.handler(stdout, stderr)
		content := base64.StdEncoding.EncodeToString(stdout.Bytes())

		rw.Write([]byte(fmt.Sprintf(`
			<env:Envelope xmlns:env="http://www.w3.org/2003/05/soap-envelope" xmlns:rsp="http://schemas.microsoft.com/wbem/wsman/1/windows/shell">
				<rsp:ReceiveResponse>
					<rsp:Stream Name="stdout" CommandId="456">%s</rsp:Stream>
					<rsp:Stream Name="stdout" CommandId="456" End="true"></rsp:Stream>
					<rsp:Stream Name="stderr" CommandId="456" End="true"></rsp:Stream>
					<rsp:CommandState State="http://schemas.microsoft.com/wbem/wsman/1/windows/shell/CommandState/Done">
						<rsp:ExitCode>%d</rsp:ExitCode>
					</rsp:CommandState>
				</rsp:ReceiveResponse>
			</env:Envelope>`, content, result)))

	} else if strings.HasSuffix(action, "transfer/Delete") {
		// end of the session
		rw.WriteHeader(http.StatusOK)
	} else {
		fmt.Printf("I don't know this action: %s\n", action)
		rw.WriteHeader(http.StatusInternalServerError)
	}
}

func readAction(env *xmlpath.Node) string {
	xpath, err := xmlpath.CompileWithNamespace(
		"//a:Action", soap.GetAllNamespaces())

	if err != nil {
		return ""
	}

	action, _ := xpath.String(env)
	return action
}

func readCommand(env *xmlpath.Node) string {
	xpath, err := xmlpath.CompileWithNamespace(
		"//rsp:Command", soap.GetAllNamespaces())

	if err != nil {
		return ""
	}

	command, _ := xpath.String(env)
	if unquoted, err := strconv.Unquote(command); err == nil {
		return unquoted
	}
	return command
}

func readCommandIdFromDesiredStream(env *xmlpath.Node) string {
	xpath, err := xmlpath.CompileWithNamespace(
		"//rsp:DesiredStream/@CommandId", soap.GetAllNamespaces())

	if err != nil {
		return ""
	}

	id, _ := xpath.String(env)
	return id
}
