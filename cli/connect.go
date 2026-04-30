package cli

import (
	"errors"
	"net/url"

	"github.com/arelate/theo/data"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

func ConnectHandler(u *url.URL) error {

	q := u.Query()

	urlStr := q.Get("url")

	username := q.Get("username")
	password := q.Get("password")

	var origin data.Origin

	if q.Has("steam") {
		origin = data.SteamOrigin
	} else if q.Has("epic-games") {
		origin = data.EpicGamesOrigin
	} else {
		origin = data.VangoghOrigin
	}

	cookie := q.Get("cookie")

	reset := q.Has("reset")

	return Connect(urlStr, username, password, cookie, origin, reset)
}

func Connect(urlStr, username, password, cookie string, origin data.Origin, reset bool) error {

	ca := nod.Begin("setting up theo connection...")
	defer ca.Done()

	rdx, err := redux.NewWriter(data.AbsReduxDir(), data.AllProperties()...)
	if err != nil {
		return err
	}

	switch origin {
	case data.VangoghOrigin:
		return vangoghSetupConnection(urlStr, username, password, rdx, reset)
	case data.SteamOrigin:
		if password != "" {
			return errors.New("steam password will be requested by SteamCMD")
		}
		return steamSetupConnection(username, rdx, reset)
	case data.EpicGamesOrigin:
		return egsSetupConnection(cookie, reset)
	default:
		return origin.ErrUnsupportedOrigin()
	}
}
