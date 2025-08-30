package handlers

import (
	"fmt"
	"html"
	"net/http"
	"strconv"
	"time"

	"github.com/Neat-Snap/blueprint-backend/config"
	"github.com/Neat-Snap/blueprint-backend/db"
	"github.com/Neat-Snap/blueprint-backend/logger"
	mw "github.com/Neat-Snap/blueprint-backend/middleware"
	"github.com/Neat-Snap/blueprint-backend/utils"
	"github.com/Neat-Snap/blueprint-backend/utils/email"
)

type FeedbackAPI struct {
	logger      logger.MultiLogger
	conn        *db.Connection
	emailClient *email.EmailClient
	cfg         config.Config
}

func NewFeedbackAPI(l logger.MultiLogger, c *db.Connection, e *email.EmailClient, cfg config.Config) *FeedbackAPI {
	return &FeedbackAPI{logger: l, conn: c, emailClient: e, cfg: cfg}
}

func (h *FeedbackAPI) SubmitEndpoint(w http.ResponseWriter, r *http.Request) {
	userObj := r.Context().Value(mw.UserObjectContextKey).(*db.User)

	var req struct {
		Message string `json:"message"`
	}
	if err := utils.ReadJSON(r.Body, w, h.logger, &req); err != nil {
		utils.WriteError(w, h.logger, err, "failed to read request body", http.StatusBadRequest)
		return
	}
	if req.Message == "" {
		utils.WriteError(w, h.logger, nil, "message is required", http.StatusBadRequest)
		return
	}

	{
		key := fmt.Sprintf("limit:feedback:%d:%s", userObj.ID, time.Now().UTC().Format("2006-01-02"))
		pipe := h.emailClient.R.R.TxPipeline()
		incr := pipe.Incr(r.Context(), key)
		pipe.ExpireNX(r.Context(), key, 24*time.Hour)
		if _, err := pipe.Exec(r.Context()); err != nil {
			utils.WriteError(w, h.logger, err, "failed to apply rate limit", http.StatusInternalServerError)
			return
		}
		if incr.Val() > 1 {
			ttl, _ := h.emailClient.R.R.TTL(r.Context(), key).Result()
			h.logger.Info("feedback rate limit reached", "user_id", userObj.ID, "ttl", ttl)
			utils.WriteError(w, h.logger, nil, "daily feedback limit reached", http.StatusTooManyRequests)
			return
		}
	}

	uname := ""
	if userObj.Name != nil {
		uname = *userObj.Name
	}
	uemail := ""
	if userObj.Email != nil {
		uemail = *userObj.Email
	}

	subject := "User Feedback"
	body := "<p>You received new feedback.</p>" +
		"<p><strong>User:</strong> " + html.EscapeString(uname) + " (" + html.EscapeString(uemail) + ")</p>" +
		"<p><strong>User ID:</strong> " + strconv.Itoa(int(userObj.ID)) + "</p>" +
		"<hr/><pre style=\"white-space:pre-wrap; font-family: ui-monospace, SFMono-Regular, Menlo, monospace\">" + html.EscapeString(req.Message) + "</pre>"

	recipient := h.cfg.SUPPORT_EMAIL
	if h.cfg.DEVELOPER_EMAIL != "" {
		recipient = h.cfg.DEVELOPER_EMAIL
	}

	h.logger.Debug("sending feedback email to %s", recipient)

	if _, err := h.emailClient.SendEmail(recipient, subject, body); err != nil {
		utils.WriteError(w, h.logger, err, "failed to send feedback", http.StatusInternalServerError)
		return
	}

	utils.WriteSuccess(w, h.logger, map[string]any{"status": "ok"}, http.StatusOK)
}
