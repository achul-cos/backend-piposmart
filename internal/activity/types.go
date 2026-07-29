package activity

import (
	"database/sql"
	"time"
)

const (
	RoleAdmin      = "ADMIN"
	RoleSupervisor = "SUPERVISOR"
	RoleSales      = "SALES"

	InteractionCall = "CALL"
	InteractionChat = "CHAT"

	StageNew       = "NEW"
	StagePossible  = "POSSIBLE"
	StagePotential = "POTENTIAL"
	StageClosing   = "CLOSING"
	StageInvalid   = "INVALID"

	StatusOpen    = "OPEN"
	StatusInvalid = "INVALID"

	TrainingOnline  = "ONLINE"
	TrainingOffline = "OFFLINE"

	TrainingScheduled = "SCHEDULED"
	TrainingCompleted = "COMPLETED"
	TrainingCanceled  = "CANCELED"

	AssignmentInvalidatedBySales = "INVALIDATED_BY_SALES"
)

type LeadState struct {
	ID                 int64
	OwnerID            sql.NullInt64
	OutletID           sql.NullInt64
	ActiveSalesID      sql.NullInt64
	CurrentOwnerUserID sql.NullInt64
	CurrentOwnerRole   string
	SupervisorID       sql.NullInt64
	Stage              string
	Status             string
	CurrentScore       sql.NullInt64
}

type RemarkReason struct {
	ID                  int64
	Score               int64
	Code                string
	Label               string
	DefaultFollowUpDays sql.NullInt64
	ReleasesAssignment  bool
}

type CustomerInteraction struct {
	ID               int64
	LeadID           sql.NullInt64
	OwnerID          sql.NullInt64
	OutletID         sql.NullInt64
	SalesID          sql.NullInt64
	SalesName        sql.NullString
	SupervisorID     sql.NullInt64
	SupervisorName   sql.NullString
	InteractionType  string
	InteractionAt    time.Time
	ContactName      sql.NullString
	ContactPhone     sql.NullString
	DurationSeconds  sql.NullInt64
	RemarkReasonID   sql.NullInt64
	RemarkScore      sql.NullInt64
	RemarkCode       sql.NullString
	RemarkLabel      sql.NullString
	Note             sql.NullString
	CustomerResponse sql.NullString
	FollowUpAt       sql.NullTime
	FollowUpNote     sql.NullString
	StageBefore      sql.NullString
	StageAfter       sql.NullString
	StatusBefore     sql.NullString
	StatusAfter      sql.NullString
	ScoreBefore      sql.NullInt64
	ScoreAfter       sql.NullInt64
	CreatedByUserID  sql.NullInt64
	CreatedByName    sql.NullString
	CreatedAt        time.Time
	UpdatedAt        time.Time
}

type LeadStageHistory struct {
	ID              int64
	LeadID          sql.NullInt64
	OwnerID         sql.NullInt64
	FromStage       sql.NullString
	ToStage         string
	FromStatus      sql.NullString
	ToStatus        string
	FromScore       sql.NullInt64
	ToScore         sql.NullInt64
	ChangedByUserID sql.NullInt64
	ChangedByName   sql.NullString
	SourceType      string
	SourceID        sql.NullInt64
	Reason          sql.NullString
	CreatedAt       time.Time
}

type TrainingReport struct {
	ID              int64
	LeadID          sql.NullInt64
	LeadCode        sql.NullString
	OwnerID         sql.NullInt64
	OwnerCode       sql.NullString
	OwnerName       sql.NullString
	OutletID        sql.NullInt64
	SalesID         sql.NullInt64
	SalesName       sql.NullString
	SupervisorID    sql.NullInt64
	SupervisorName  sql.NullString
	TrainingType    string
	Status          string
	ScheduledAt     time.Time
	CompletedAt     sql.NullTime
	CanceledAt      sql.NullTime
	RescheduledAt   sql.NullTime
	Location        sql.NullString
	MeetingURL      sql.NullString
	Note            sql.NullString
	ResultNote      sql.NullString
	CancelReason    sql.NullString
	CreatedByUserID sql.NullInt64
	CreatedByName   sql.NullString
	UpdatedByUserID sql.NullInt64
	UpdatedByName   sql.NullString
	CreatedAt       time.Time
	UpdatedAt       time.Time
}

type CreateInteractionRequest struct {
	Type             string     `json:"type" binding:"required"`
	InteractionAt    *time.Time `json:"interaction_at"`
	ContactName      string     `json:"contact_name"`
	ContactPhone     string     `json:"contact_phone"`
	DurationSeconds  *int64     `json:"duration_seconds"`
	RemarkReasonID   *int64     `json:"remark_reason_id"`
	RemarkScore      *int64     `json:"remark_score"`
	Note             string     `json:"note"`
	CustomerResponse string     `json:"customer_response"`
	FollowUpAt       *time.Time `json:"follow_up_at"`
	FollowUpNote     string     `json:"follow_up_note"`
}

type ScheduleTrainingRequest struct {
	TrainingType    string    `json:"training_type" binding:"required"`
	ScheduledAt     time.Time `json:"scheduled_at" binding:"required"`
	Location        string    `json:"location"`
	MeetingURL      string    `json:"meeting_url"`
	Note            string    `json:"note"`
}

type RescheduleTrainingRequest struct {
	ScheduledAt time.Time `json:"scheduled_at" binding:"required"`
	Note        string    `json:"note"`
}

type CompleteTrainingRequest struct {
	CompletedAt *time.Time `json:"completed_at"`
	ResultNote  string     `json:"result_note"`
}

type CancelTrainingRequest struct {
	CancelReason string `json:"cancel_reason"`
}

type InteractionListParams struct {
	LeadID          *int64
	Type            string
	Score           *int64
	SalesID         *int64
	InteractionFrom *time.Time
	InteractionTo   *time.Time
	FollowUpFrom    *time.Time
	FollowUpTo      *time.Time
	OnlyFollowUps   bool
	All             bool
	Page            int
	Limit           int
	Sort            string
}

type TrainingListParams struct {
	LeadID        *int64
	Status        string
	TrainingType  string
	SalesID       *int64
	ScheduledFrom *time.Time
	ScheduledTo   *time.Time
	All           bool
	Page          int
	Limit         int
	Sort          string
}

type PaginationMeta struct {
	Page  int   `json:"page"`
	Limit int   `json:"limit"`
	Total int64 `json:"total"`
}

type CustomerInteractionResponse struct {
	ID               int64      `json:"id"`
	LeadID           *int64     `json:"lead_id,omitempty"`
	OwnerID          *int64     `json:"owner_id,omitempty"`
	OutletID         *int64     `json:"outlet_id,omitempty"`
	Sales            *UserBrief `json:"sales,omitempty"`
	Supervisor       *UserBrief `json:"supervisor,omitempty"`
	Type             string     `json:"type"`
	InteractionAt    time.Time  `json:"interaction_at"`
	ContactName      string     `json:"contact_name,omitempty"`
	ContactPhone     string     `json:"contact_phone,omitempty"`
	DurationSeconds  *int64     `json:"duration_seconds,omitempty"`
	RemarkReasonID   *int64     `json:"remark_reason_id,omitempty"`
	RemarkScore      *int64     `json:"remark_score,omitempty"`
	RemarkCode       string     `json:"remark_code,omitempty"`
	RemarkLabel      string     `json:"remark_label,omitempty"`
	Note             string     `json:"note,omitempty"`
	CustomerResponse string     `json:"customer_response,omitempty"`
	FollowUpAt       *time.Time `json:"follow_up_at,omitempty"`
	FollowUpNote     string     `json:"follow_up_note,omitempty"`
	StageBefore      string     `json:"stage_before,omitempty"`
	StageAfter       string     `json:"stage_after,omitempty"`
	StatusBefore     string     `json:"status_before,omitempty"`
	StatusAfter      string     `json:"status_after,omitempty"`
	ScoreBefore      *int64     `json:"score_before,omitempty"`
	ScoreAfter       *int64     `json:"score_after,omitempty"`
	CreatedBy        *UserBrief `json:"created_by,omitempty"`
	CreatedAt        time.Time  `json:"created_at"`
	UpdatedAt        time.Time  `json:"updated_at"`
}

type StageHistoryResponse struct {
	ID         int64      `json:"id"`
	LeadID     *int64     `json:"lead_id,omitempty"`
	OwnerID    *int64     `json:"owner_id,omitempty"`
	FromStage  string     `json:"from_stage,omitempty"`
	ToStage    string     `json:"to_stage"`
	FromStatus string     `json:"from_status,omitempty"`
	ToStatus   string     `json:"to_status"`
	FromScore  *int64     `json:"from_score,omitempty"`
	ToScore    *int64     `json:"to_score,omitempty"`
	ChangedBy  *UserBrief `json:"changed_by,omitempty"`
	SourceType string     `json:"source_type"`
	SourceID   *int64     `json:"source_id,omitempty"`
	Reason     string     `json:"reason,omitempty"`
	CreatedAt  time.Time  `json:"created_at"`
}

type TrainingReportResponse struct {
	ID              int64      `json:"id"`
	LeadID          *int64     `json:"lead_id,omitempty"`
	LeadCode        string     `json:"lead_code,omitempty"`
	OwnerID         *int64     `json:"owner_id,omitempty"`
	OwnerCode       string     `json:"owner_code,omitempty"`
	OwnerName       string     `json:"owner_name,omitempty"`
	OutletID        *int64     `json:"outlet_id,omitempty"`
	Sales           *UserBrief `json:"sales,omitempty"`
	Supervisor      *UserBrief `json:"supervisor,omitempty"`
	TrainingType    string     `json:"training_type"`
	Status          string     `json:"status"`
	ScheduledAt     time.Time  `json:"scheduled_at"`
	CompletedAt     *time.Time `json:"completed_at,omitempty"`
	CanceledAt      *time.Time `json:"canceled_at,omitempty"`
	RescheduledAt   *time.Time `json:"rescheduled_at,omitempty"`
	Location        string     `json:"location,omitempty"`
	MeetingURL      string     `json:"meeting_url,omitempty"`
	Note            string     `json:"note,omitempty"`
	ResultNote      string     `json:"result_note,omitempty"`
	CancelReason    string     `json:"cancel_reason,omitempty"`
	CreatedBy       *UserBrief `json:"created_by,omitempty"`
	UpdatedBy       *UserBrief `json:"updated_by,omitempty"`
	CreatedAt       time.Time  `json:"created_at"`
	UpdatedAt       time.Time  `json:"updated_at"`
}

type UserBrief struct {
	ID   int64  `json:"id"`
	Name string `json:"name"`
	Role string `json:"role,omitempty"`
}

type InteractionListResponse struct {
	Items      []CustomerInteractionResponse `json:"items"`
	Pagination PaginationMeta                `json:"pagination"`
}

type StageHistoryListResponse struct {
	Items []StageHistoryResponse `json:"items"`
	Total int                    `json:"total"`
}

type TrainingListResponse struct {
	Items      []TrainingReportResponse `json:"items"`
	Pagination PaginationMeta           `json:"pagination"`
}

func NewInteractionResponse(item CustomerInteraction) CustomerInteractionResponse {
	return CustomerInteractionResponse{
		ID:               item.ID,
		LeadID:           nullableInt64Ptr(item.LeadID),
		OwnerID:          nullableInt64Ptr(item.OwnerID),
		OutletID:         nullableInt64Ptr(item.OutletID),
		Sales:            nullableUser(item.SalesID, item.SalesName, RoleSales),
		Supervisor:       nullableUser(item.SupervisorID, item.SupervisorName, RoleSupervisor),
		Type:             item.InteractionType,
		InteractionAt:    item.InteractionAt,
		ContactName:      item.ContactName.String,
		ContactPhone:     item.ContactPhone.String,
		DurationSeconds:  nullableInt64Ptr(item.DurationSeconds),
		RemarkReasonID:   nullableInt64Ptr(item.RemarkReasonID),
		RemarkScore:      nullableInt64Ptr(item.RemarkScore),
		RemarkCode:       item.RemarkCode.String,
		RemarkLabel:      item.RemarkLabel.String,
		Note:             item.Note.String,
		CustomerResponse: item.CustomerResponse.String,
		FollowUpAt:       nullableTimePtr(item.FollowUpAt),
		FollowUpNote:     item.FollowUpNote.String,
		StageBefore:      item.StageBefore.String,
		StageAfter:       item.StageAfter.String,
		StatusBefore:     item.StatusBefore.String,
		StatusAfter:      item.StatusAfter.String,
		ScoreBefore:      nullableInt64Ptr(item.ScoreBefore),
		ScoreAfter:       nullableInt64Ptr(item.ScoreAfter),
		CreatedBy:        nullableUser(item.CreatedByUserID, item.CreatedByName, ""),
		CreatedAt:        item.CreatedAt,
		UpdatedAt:        item.UpdatedAt,
	}
}

func NewStageHistoryResponse(item LeadStageHistory) StageHistoryResponse {
	return StageHistoryResponse{
		ID:         item.ID,
		LeadID:     nullableInt64Ptr(item.LeadID),
		OwnerID:    nullableInt64Ptr(item.OwnerID),
		FromStage:  item.FromStage.String,
		ToStage:    item.ToStage,
		FromStatus: item.FromStatus.String,
		ToStatus:   item.ToStatus,
		FromScore:  nullableInt64Ptr(item.FromScore),
		ToScore:    nullableInt64Ptr(item.ToScore),
		ChangedBy:  nullableUser(item.ChangedByUserID, item.ChangedByName, ""),
		SourceType: item.SourceType,
		SourceID:   nullableInt64Ptr(item.SourceID),
		Reason:     item.Reason.String,
		CreatedAt:  item.CreatedAt,
	}
}

func NewTrainingReportResponse(item TrainingReport) TrainingReportResponse {
	return TrainingReportResponse{
		ID:              item.ID,
		LeadID:          nullableInt64Ptr(item.LeadID),
		LeadCode:        item.LeadCode.String,
		OwnerID:         nullableInt64Ptr(item.OwnerID),
		OwnerCode:       item.OwnerCode.String,
		OwnerName:       item.OwnerName.String,
		OutletID:        nullableInt64Ptr(item.OutletID),
		Sales:           nullableUser(item.SalesID, item.SalesName, RoleSales),
		Supervisor:      nullableUser(item.SupervisorID, item.SupervisorName, RoleSupervisor),
		TrainingType:    item.TrainingType,
		Status:          item.Status,
		ScheduledAt:     item.ScheduledAt,
		CompletedAt:     nullableTimePtr(item.CompletedAt),
		CanceledAt:      nullableTimePtr(item.CanceledAt),
		RescheduledAt:   nullableTimePtr(item.RescheduledAt),
		Location:        item.Location.String,
		MeetingURL:      item.MeetingURL.String,
		Note:            item.Note.String,
		ResultNote:      item.ResultNote.String,
		CancelReason:    item.CancelReason.String,
		CreatedBy:       nullableUser(item.CreatedByUserID, item.CreatedByName, ""),
		UpdatedBy:       nullableUser(item.UpdatedByUserID, item.UpdatedByName, ""),
		CreatedAt:       item.CreatedAt,
		UpdatedAt:       item.UpdatedAt,
	}
}

func nullableInt64Ptr(value sql.NullInt64) *int64 {
	if !value.Valid {
		return nil
	}
	out := value.Int64
	return &out
}

func nullableTimePtr(value sql.NullTime) *time.Time {
	if !value.Valid {
		return nil
	}
	out := value.Time
	return &out
}

func nullableUser(id sql.NullInt64, name sql.NullString, role string) *UserBrief {
	if !id.Valid {
		return nil
	}
	return &UserBrief{ID: id.Int64, Name: name.String, Role: role}
}
