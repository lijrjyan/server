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

package subject

import (
	"net/http"

	"github.com/labstack/echo/v4"
	"github.com/trim21/errgo"

	"github.com/bangumi/server/internal/episode"
	"github.com/bangumi/server/internal/model"
	"github.com/bangumi/server/internal/pkg/null"
	"github.com/bangumi/server/internal/subject"
	"github.com/bangumi/server/web/accessor"
	"github.com/bangumi/server/web/req"
	"github.com/bangumi/server/web/res"
)

func (h Subject) BatchGet(c echo.Context) error {
	var payload req.SubjectBatchRequest
	if err := c.Echo().JSONSerializer.Deserialize(c, &payload); err != nil {
		return res.JSONError(c, err)
	}

	if err := payload.Validate(); err != nil {
		return err
	}

	ctx := c.Request().Context()

	order := payload.IDs
	uniqueIDs := make([]model.SubjectID, 0, len(order))
	seen := make(map[model.SubjectID]struct{}, len(order))
	for _, id := range order {
		if _, ok := seen[id]; ok {
			continue
		}
		seen[id] = struct{}{}
		uniqueIDs = append(uniqueIDs, id)
	}

	u := accessor.GetFromCtx(c)
	filter := subject.Filter{NSFW: null.Bool{Value: false, Set: !u.AllowNSFW()}}

	subjects, err := h.subject.GetByIDs(ctx, uniqueIDs, filter)
	if err != nil {
		return errgo.Wrap(err, "subject.GetByIDs")
	}

	tags, err := h.tag.GetByIDs(ctx, uniqueIDs)
	if err != nil {
		return errgo.Wrap(err, "tag.GetByIDs")
	}

	episodeTotals := make(map[model.SubjectID]int64, len(subjects))
	for _, id := range uniqueIDs {
		subject, ok := subjects[id]
		if !ok || subject.Redirect != 0 {
			continue
		}
		total, err := h.episode.Count(ctx, id, episode.Filter{})
		if err != nil {
			return errgo.Wrap(err, "episode.Count")
		}
		episodeTotals[id] = total
	}

	data := make([]res.SubjectV0, 0, len(order))
	missingSet := make(map[model.SubjectID]struct{})
	missing := make([]model.SubjectID, 0)
	var redirects map[model.SubjectID]model.SubjectID

	for _, id := range order {
		subject, ok := subjects[id]
		if !ok {
			if _, seenMissing := missingSet[id]; !seenMissing {
				missing = append(missing, id)
				missingSet[id] = struct{}{}
			}
			continue
		}

		if subject.Redirect != 0 {
			if redirects == nil {
				redirects = make(map[model.SubjectID]model.SubjectID)
			}
			redirects[id] = subject.Redirect
			continue
		}

		metaTags := tags[id]
		totalEpisodes := episodeTotals[id]
		data = append(data, res.ToSubjectV0(subject, totalEpisodes, metaTags))
	}

	response := res.SubjectBatch{
		Data:      data,
		Missing:   missing,
		Redirects: redirects,
	}

	return c.JSON(http.StatusOK, response)
}
