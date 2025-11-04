// SPDX-License-Identifier: AGPL-3.0-only
//
// This program is free software: you can redistribute it and/or modify
// it under the terms of the GNU Affero General Public License as published
// by the Free Software Foundation, version 3.
//
// This program is distributed in the hope that it will be useful,
// but WITHOUT ANY WARRANTY; without even the implied warranty of
// MERCHANTABILITY or FITNESS FOR A PARTICULAR PURPOSE.
// See the GNU Affero General Public License for more details.
//
// You should have received a copy of the GNU Affero General Public License
// along with this program. If not, see <https://www.gnu.org/licenses/>

package req

import (
	"fmt"

	"github.com/bangumi/server/internal/model"
	"github.com/bangumi/server/web/res"
)

const SubjectBatchMaxSize = 50

type SubjectBatchRequest struct {
	IDs []model.SubjectID `json:"ids"`
}

func (r SubjectBatchRequest) Validate() error {
	if len(r.IDs) == 0 {
		return res.BadRequest("ids is required")
	}

	if len(r.IDs) > SubjectBatchMaxSize {
		return res.BadRequest(fmt.Sprintf("最多允许 %d 个条目 ID", SubjectBatchMaxSize))
	}

	for _, id := range r.IDs {
		if id == 0 {
			return res.BadRequest("ids must be a positive integer")
		}
	}

	return nil
}
