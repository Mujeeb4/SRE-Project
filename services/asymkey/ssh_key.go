// Copyright 2021 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package asymkey

import (
	asymkey_model "code.gitea.io/gitea/models/asymkey"
	"code.gitea.io/gitea/models/db"
	user_model "code.gitea.io/gitea/models/user"
	"code.gitea.io/gitea/services/audit"
)

// DeletePublicKey deletes SSH key information both in database and authorized_keys file.
func DeletePublicKey(doer *user_model.User, id int64) (err error) {
	key, err := asymkey_model.GetPublicKeyByID(id)
	if err != nil {
		return err
	}

	owner, err := user_model.GetUserByID(db.DefaultContext, key.OwnerID)
	if err != nil {
		return err
	}

	// Check if user has access to delete this key.
	if !doer.IsAdmin && doer.ID != key.OwnerID {
		return asymkey_model.ErrKeyAccessDenied{
			UserID: doer.ID,
			KeyID:  key.ID,
			Note:   "public",
		}
	}

	ctx, committer, err := db.TxContext(db.DefaultContext)
	if err != nil {
		return err
	}
	defer committer.Close()

	if err = asymkey_model.DeletePublicKeys(ctx, id); err != nil {
		return err
	}

	if err = committer.Commit(); err != nil {
		return err
	}
	committer.Close()

	if key.Type == asymkey_model.KeyTypePrincipal {
		audit.Record(audit.UserKeyPrincipalRemove, doer, owner, key, "Removed principal key %s.", key.Name)

		return asymkey_model.RewriteAllPrincipalKeys(db.DefaultContext)
	}

	audit.Record(audit.UserKeySSHRemove, doer, owner, key, "Removed SSH key %s.", key.Fingerprint)

	return asymkey_model.RewriteAllPublicKeys()
}
