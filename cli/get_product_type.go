package cli

import (
	"errors"
	"fmt"
	"io"
	"net/http"
	"path"

	"github.com/arelate/southern_light/vangogh_integration"
	"github.com/arelate/theo/data"
	"github.com/boggydigital/kevlar"
	"github.com/boggydigital/nod"
	"github.com/boggydigital/redux"
)

func productTypeRequest(id string, pt vangogh_integration.ProductType, rdx redux.Readable) (*http.Request, error) {
	switch pt {
	case vangogh_integration.GogChecksums:
		apiGogChecksumsPath := path.Join(data.ApiGogChecksumsPath, id)
		return data.VangoghApiRequest(http.MethodGet, apiGogChecksumsPath, nil, rdx)
	case vangogh_integration.GogFilenames:
		apiGogFilenamesPath := path.Join(data.ApiGogFilenamesPath, id)
		return data.VangoghApiRequest(http.MethodGet, apiGogFilenamesPath, nil, rdx)
	case vangogh_integration.GogDetails:
		apiMetadataGogDetailsPath := path.Join(data.ApiMetadataPath, pt.String(), id)
		return data.VangoghApiRequest(http.MethodGet, apiMetadataGogDetailsPath, nil, rdx)
	case vangogh_integration.GogApiProducts:
		apiMetadataGogApiProductPath := path.Join(data.ApiMetadataPath, pt.String(), id)
		return data.VangoghApiRequest(http.MethodGet, apiMetadataGogApiProductPath, nil, rdx)
	case vangogh_integration.AvailableProducts:
		return data.VangoghApiRequest(http.MethodGet, data.ApiAvailableProductsPath, nil, rdx)
	default:
		return nil, errors.New("http request not supported for " + pt.String())
	}
}

func getProductType(id string, pt vangogh_integration.ProductType, rdx redux.Writeable, force bool) (io.ReadCloser, error) {

	gpta := nod.Begin(" getting %s %s...", pt, id)
	defer gpta.Done()

	kvPt, err := kevlar.New(vangogh_integration.AbsProductTypeDir(pt), pt.Ext())
	if err != nil {
		return nil, err
	}

	if kvPt.Has(id) && !force {

		var rcLocal io.ReadCloser
		rcLocal, err = kvPt.Get(id)
		if err != nil {
			return nil, err
		}

		gpta.EndWithResult("read local")
		return rcLocal, nil
	}

	if err = vangoghValidateSessionToken(rdx); err != nil {
		return nil, err
	}

	if err = fetchRemoteProductType(id, pt, rdx, kvPt); err != nil {
		return nil, err
	}

	if err = reduceProductType(id, pt, rdx, kvPt); err != nil {
		return nil, err
	}

	gpta.EndWithResult("fetched remote")

	return kvPt.Get(id)
}

func fetchRemoteProductType(id string, pt vangogh_integration.ProductType, rdx redux.Readable, kvPt kevlar.KeyValues) error {

	frpta := nod.Begin(" fetching remote %s %s...", pt, id)
	defer frpta.Done()

	req, err := productTypeRequest(id, pt, rdx)
	if err != nil {
		return err
	}

	resp, err := http.DefaultClient.Do(req)
	if err != nil {
		return err
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode > 299 {
		return errors.New(fmt.Sprintf("error fetching %s: %s", pt, resp.Status))
	}

	return kvPt.Set(id, resp.Body)
}

func reduceProductType(id string, pt vangogh_integration.ProductType, rdx redux.Writeable, kvPt kevlar.KeyValues) error {

	rpta := nod.Begin(" reducing %s...", pt)
	defer rpta.Done()

	switch pt {
	case vangogh_integration.GogDetails:
		return vangoghReduceGogDetails(id, kvPt, rdx)
	case vangogh_integration.GogChecksums:
		fallthrough
	case vangogh_integration.GogFilenames:
		fallthrough
	case vangogh_integration.GogApiProducts:
		fallthrough
	case vangogh_integration.AvailableProducts:
		return nil
	default:
		return errors.New("reduction not supported for " + pt.String())
	}
}
