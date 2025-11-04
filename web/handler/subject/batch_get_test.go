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

package subject_test

import (
	"bytes"
	"encoding/json"
	"net/http"
	"net/http/httptest"
	"testing"

	"github.com/labstack/echo/v4"
	"github.com/stretchr/testify/mock"
	"github.com/stretchr/testify/require"

	"github.com/bangumi/server/internal/episode"
	"github.com/bangumi/server/internal/mocks"
	"github.com/bangumi/server/internal/model"
	"github.com/bangumi/server/internal/pkg/null"
	"github.com/bangumi/server/internal/pkg/test"
	"github.com/bangumi/server/internal/subject"
	"github.com/bangumi/server/internal/tag"
	"github.com/bangumi/server/web/req"
	"github.com/bangumi/server/web/res"
)

func TestSubject_BatchGet_Success(t *testing.T) {
	t.Parallel()

	subjectRepo := mocks.NewSubjectCachedRepo(t)
	subjectRepo.EXPECT().
		GetByIDs(mock.Anything, []model.SubjectID{1, 2, 3}, subject.Filter{NSFW: null.NewBool(false)}).
		Return(map[model.SubjectID]model.Subject{
			1: {ID: 1, Name: "subject-1", Tags: []model.Tag{{Name: "tag-1"}}},
			2: {ID: 2, Redirect: 5},
		}, nil)

	tagRepo := mocks.NewTagRepo(t)
	tagRepo.EXPECT().
		GetByIDs(mock.Anything, []model.SubjectID{1, 2, 3}).
		Return(map[model.SubjectID][]tag.Tag{
			1: {{Name: "meta", Count: 1}},
		}, nil)

	episodeRepo := mocks.NewEpisodeRepo(t)
	episodeRepo.EXPECT().
		Count(mock.Anything, model.SubjectID(1), episode.Filter{}).
		Return(int64(12), nil)

	app := test.GetWebApp(t, test.Mock{
		SubjectCachedRepo: subjectRepo,
		TagRepo:           tagRepo,
		EpisodeRepo:       episodeRepo,
	})

	body, err := json.Marshal(map[string]any{
		"ids": []model.SubjectID{1, 2, 3, 1},
	})
	require.NoError(t, err)

	req := httptest.NewRequest(http.MethodPost, "/v0/subjects", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	require.Equal(t, http.StatusOK, rec.Code)

	var rsp res.SubjectBatch
	require.NoError(t, json.Unmarshal(rec.Body.Bytes(), &rsp))

	require.Len(t, rsp.Data, 2)
	require.EqualValues(t, 1, rsp.Data[0].ID)
	require.EqualValues(t, 1, rsp.Data[1].ID)
	require.Equal(t, int64(12), rsp.Data[0].TotalEpisodes)
	require.Equal(t, []model.SubjectID{3}, rsp.Missing)
	require.Equal(t, map[model.SubjectID]model.SubjectID{2: 5}, rsp.Redirects)
}

func TestSubject_BatchGet_Validation(t *testing.T) {
	t.Parallel()

	ids := make([]model.SubjectID, req.SubjectBatchMaxSize+1)
	for i := range ids {
		ids[i] = model.SubjectID(i + 1)
	}

	body, err := json.Marshal(map[string]any{"ids": ids})
	require.NoError(t, err)

	app := test.GetWebApp(t, test.Mock{})

	req := httptest.NewRequest(http.MethodPost, "/v0/subjects", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}

func TestSubject_BatchGet_InvalidID(t *testing.T) {
	t.Parallel()

	body, err := json.Marshal(map[string]any{"ids": []int{1, 0}})
	require.NoError(t, err)

	app := test.GetWebApp(t, test.Mock{})

	req := httptest.NewRequest(http.MethodPost, "/v0/subjects", bytes.NewReader(body))
	req.Header.Set(echo.HeaderContentType, echo.MIMEApplicationJSON)
	rec := httptest.NewRecorder()

	app.ServeHTTP(rec, req)

	require.Equal(t, http.StatusBadRequest, rec.Code)
}
