package cli

import (
	"net/url"

	"github.com/arelate/theo/data"
	"github.com/boggydigital/camino"
	"github.com/boggydigital/nod"
)

func BackupMetadataHandler(_ *url.URL) error {
	return BackupMetadata()
}

func BackupMetadata() error {

	ba := nod.NewProgress("backing up local metadata...")
	defer ba.Done()

	if err := camino.Compress(data.Metadata, data.Backups); err != nil {
		return err
	}

	ca := nod.NewProgress("cleaning up old backups...")
	defer ca.Done()

	if err := camino.CleanupTimed(data.Backups, true); err != nil {
		return err
	}

	return nil
}
