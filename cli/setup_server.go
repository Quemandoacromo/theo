package cli

import (
	"errors"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/pathways"
	"github.com/boggydigital/redux"
	"io"
	"net/http"
	"net/url"
)

type SetupProperties map[string]string

func SetupServerHandler(u *url.URL) error {

	q := u.Query()

	protocol := q.Get("protocol")
	address := q.Get("address")
	port := q.Get("port")

	username := q.Get("username")
	password := q.Get("password")

	reset := q.Has("reset")
	test := q.Has("test")

	return SetupServer(protocol, address, port, username, password, reset, test)
}

func SetupServer(
	protocol, address, port string,
	username, password string,
	reset, test bool) error {

	if reset {
		if err := resetServerSetup(); err != nil {
			return err
		}
	}

	sa := nod.Begin("setting up server connection...")
	defer sa.Done()

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewWriter(reduxDir, data.ServerConnectionProperties)
	if err != nil {
		return err
	}

	setupProperties := make(map[string][]string)

	if protocol != "" {
		setupProperties[data.ServerProtocolProperty] = []string{protocol}
	}

	if address != "" {
		setupProperties[data.ServerAddressProperty] = []string{address}
	}

	if port != "" {
		setupProperties[data.ServerPortProperty] = []string{port}
	}

	if username != "" {
		setupProperties[data.ServerUsernameProperty] = []string{username}
	}

	if password != "" {
		setupProperties[data.ServerPasswordProperty] = []string{password}
	}

	if len(setupProperties) > 0 {
		if err = rdx.BatchReplaceValues(data.ServerConnectionProperties, setupProperties); err != nil {
			return err
		}
	}

	if test {
		return testServer()
	}

	return nil
}

func resetServerSetup() error {
	rsa := nod.Begin("resetting server setup...")
	defer rsa.Done()

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewWriter(reduxDir, data.ServerConnectionProperties)
	if err != nil {
		return err
	}

	setupProperties := []string{
		data.ServerProtocolProperty,
		data.ServerAddressProperty,
		data.ServerPortProperty,
		data.ServerUsernameProperty,
		data.ServerPasswordProperty,
	}

	if err = rdx.CutKeys(data.ServerConnectionProperties, setupProperties...); err != nil {
		return err
	}

	return nil
}

func testServer() error {

	tsa := nod.Begin("testing server connection...")
	defer tsa.EndWithResult("success - server setup is valid")

	reduxDir, err := pathways.GetAbsRelDir(data.Redux)
	if err != nil {
		return err
	}

	rdx, err := redux.NewReader(reduxDir, data.ServerConnectionProperties)
	if err != nil {
		return err
	}

	if err = testServerConnectivity(rdx); err != nil {
		return err
	}

	if err = testServerAuth(rdx); err != nil {
		return err
	}

	return nil
}

func testServerConnectivity(rdx redux.Readable) error {

	testUrl, err := data.ServerUrl(rdx, data.ApiHealthPath, nil)
	if err != nil {
		return err
	}

	tvca := nod.Begin(" testing connectivity to %s...", testUrl.String())
	defer tvca.Done()

	resp, err := http.DefaultClient.Get(testUrl.String())
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return errors.New(resp.Status)
	}

	bts, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if string(bts) != "ok" {
		return errors.New("unexpected health response")
	}

	tvca.EndWithResult("done, healthy")

	return nil
}

func testServerAuth(rdx redux.Readable) error {

	testUrl, err := data.ServerUrl(rdx, data.ApiHealthAuthPath, nil)
	if err != nil {
		return err
	}

	tvaa := nod.Begin(" testing auth for %s...", testUrl.String())
	defer tvaa.Done()

	req, err := http.NewRequest(http.MethodGet, testUrl.String(), nil)
	if err != nil {
		return err
	}

	if username, ok := rdx.GetLastVal(data.ServerConnectionProperties, data.ServerUsernameProperty); ok && username != "" {
		if password, sure := rdx.GetLastVal(data.ServerConnectionProperties, data.ServerPasswordProperty); sure && password != "" {
			req.SetBasicAuth(username, password)
		} else {
			return errors.New("password cannot be empty")
		}
	} else {
		return errors.New("username cannot be empty")
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return errors.New(resp.Status)
	}

	bts, err := io.ReadAll(resp.Body)
	if err != nil {
		return err
	}

	if string(bts) != "ok" {
		return errors.New("unexpected health-auth response")
	}

	tvaa.EndWithResult("done, healthy")

	return nil
}
