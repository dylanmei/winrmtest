package winrmtest

import (
	"github.com/masterzen/winrm/soap"
	"github.com/masterzen/xmlpath"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
)

func Test_create_shell(t *testing.T) {
	w := &wsman{}

	res := httptest.NewRecorder()
	req, _ := http.NewRequest("POST", "", strings.NewReader(`
    <env:Envelope xmlns:env="http://www.w3.org/2003/05/soap-envelope" xmlns:a="http://schemas.xmlsoap.org/ws/2004/08/addressing">
			<a:Action mustUnderstand="true">http://schemas.xmlsoap.org/ws/2004/09/transfer/Create</a:Action>
		</env:Envelope>`))

	w.ServeHTTP(res, req)
	if res.Code != http.StatusOK {
		t.Errorf("Expected 200 OK but was %d.\n", res.Code)
	}

	if contentType := res.HeaderMap.Get("Content-Type"); contentType != "application/soap+xml" {
		t.Errorf("Expected ContentType application/soap+xml was %s.\n", contentType)
	}

	env, err := xmlpath.Parse(res.Body)
	if err != nil {
		t.Error("Couldn't compile the SOAP response.")
	}

	xpath, _ := xmlpath.CompileWithNamespace(
		"//rsp:ShellId", soap.GetAllNamespaces())

	if _, found := xpath.String(env); !found {
		t.Error("Expected a Shell identifier.")
	}
}
