package utils

import (
	"fmt"
	"os"

	"github.com/argoproj-labs/argocd-autopilot/pkg/fs"

	billyUtils "github.com/go-git/go-billy/v5/util"
)

type BulkWriteRequest struct {
	Filename string
	Data     []byte
	Perm     os.FileMode
	ErrMsg   string
}

// BulkWrite accepts multiple write requests and perform them sequentially. If it encounters
// an error it stops it's operation and returns this error.
//
// default file permission is 0644.
func BulkWrite(fsys fs.FS, writes ...BulkWriteRequest) error {
	for _, req := range writes {
		var perm os.FileMode = 0644
		if req.Perm != 0 {
			perm = req.Perm
		}

		if err := billyUtils.WriteFile(fsys, req.Filename, req.Data, perm); err != nil {
			if req.ErrMsg != "" {
				return fmt.Errorf("%s: %w", req.ErrMsg, err)
			}
			return err
		}
	}
	return nil
}
