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
	funcsById   map[string]CommandFunc
	funcsByText map[string]CommandFunc
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
		fmt.Println("Creating a new shell")

		rw.Write([]byte(`
      <env:Envelope xmlns:env="http://www.w3.org/2003/05/soap-envelope" xmlns:rsp="http://schemas.microsoft.com/wbem/wsman/1/windows/shell">
        <rsp:ShellId>123</rsp:ShellId>
      </env:Envelope>
    `))

	} else if strings.HasSuffix(action, "shell/Command") {
		// execute on behalf of the client
		text := readCommand(env)
		fmt.Printf("Executing command: %s\n", text)

		handler := w.funcsByText[text]
		w.mapCommandIdToFunc("456", handler)

		rw.Write([]byte(`
      <env:Envelope xmlns:env="http://www.w3.org/2003/05/soap-envelope" xmlns:rsp="http://schemas.microsoft.com/wbem/wsman/1/windows/shell">
        <rsp:CommandId>456</rsp:CommandId>
      </env:Envelope>
    `))

	} else if strings.HasSuffix(action, "shell/Receive") {
		// client ready to receive the results

		id := readCommandIdFromDesiredStream(env)
		commandFunc := w.funcsById[id]

		stdout := bytes.NewBuffer(make([]byte, 0))
		stderr := bytes.NewBuffer(make([]byte, 0))
		result := commandFunc(stdout, stderr)
		content := base64.StdEncoding.EncodeToString(stdout.Bytes())

		rw.Write([]byte(fmt.Sprintf(`
      <env:Envelope xmlns:env="http://www.w3.org/2003/05/soap-envelope" xmlns:rsp="http://schemas.microsoft.com/wbem/wsman/1/windows/shell">
        <rsp:Stream Name="stdout" CommandId="456">%s</rsp:Stream>
        <rsp:Stream Name="stdout" CommandId="456" End="true"></rsp:Stream>
        <rsp:Stream Name="stderr" CommandId="456" End="true"></rsp:Stream>
        <rsp:CommandState State="http://schemas.microsoft.com/wbem/wsman/1/windows/shell/CommandState/Done">
          <rsp:ExitCode>%d</rsp:ExitCode>
        </rsp:CommandState>
      </env:Envelope>
    `, content, result)))

	} else if strings.HasSuffix(action, "transfer/Delete") {
		rw.WriteHeader(http.StatusOK)
	} else {
		fmt.Printf("I don't know this action: %s\n", action)
		rw.WriteHeader(http.StatusInternalServerError)
	}
}

func (w *wsman) mapCommandTextToFunc(cmd string, f CommandFunc) {
	if w.funcsByText == nil {
		w.funcsByText = make(map[string]CommandFunc)
	}
	w.funcsByText[cmd] = f
}

func (w *wsman) mapCommandIdToFunc(id string, f CommandFunc) {
	if w.funcsById == nil {
		w.funcsById = make(map[string]CommandFunc)
	}
	w.funcsById[id] = f
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
