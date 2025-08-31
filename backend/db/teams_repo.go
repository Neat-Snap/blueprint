package db

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type teamsRepo struct{ db *gorm.DB }

func (r *teamsRepo) Create(ctx context.Context, t *Team) error {
	return r.db.WithContext(ctx).Create(t).Error
}

func (r *teamsRepo) ByID(ctx context.Context, id uint) (*Team, error) {
	var team Team
	err := r.db.WithContext(ctx).
		Preload("Owner").
		Preload("Users").
		First(&team, id).Error
	return &team, err
}

func (r *teamsRepo) AddMember(ctx context.Context, teamID, userID uint, role string) error {
	uw := UserTeam{UserID: userID, TeamID: teamID, Role: role}

	return r.db.WithContext(ctx).
		Clauses(clause.OnConflict{
			Columns:   []clause.Column{{Name: "user_id"}, {Name: "team_id"}},
			DoUpdates: clause.Assignments(map[string]interface{}{"role": role}),
		}).
		Create(&uw).Error
}

func (r *teamsRepo) RemoveMember(ctx context.Context, teamID, userID uint) error {
	return r.db.WithContext(ctx).Where("user_id=? AND team_id=?", userID, teamID).
		Delete(&UserTeam{}).Error
}

func (r *teamsRepo) ListForUser(ctx context.Context, userID uint) ([]Team, error) {
	var teams []Team
	err := r.db.WithContext(ctx).
		Joins("JOIN user_teams ut ON ut.team_id = teams.id").
		Where("ut.user_id = ?", userID).
		Preload("Owner").
		Find(&teams).Error
	return teams, err
}

func (r *teamsRepo) ReassignOwner(ctx context.Context, teamID, newOwnerID uint) error {
	return r.db.WithContext(ctx).
		Model(&Team{}).
		Where("id = ?", teamID).
		Update("owner_id", newOwnerID).Error
}

func (r *teamsRepo) GetUserRole(ctx context.Context, teamID, userID uint) (string, error) {
	var uw UserTeam
	err := r.db.WithContext(ctx).
		Where("team_id = ? AND user_id = ?", teamID, userID).
		First(&uw).Error
	if err != nil {
		return "", err
	}
	return uw.Role, nil
}

func (r *teamsRepo) RolesForTeam(ctx context.Context, teamID uint) (map[uint]string, error) {
	var uws []UserTeam
	if err := r.db.WithContext(ctx).Where("team_id = ?", teamID).Find(&uws).Error; err != nil {
		return nil, err
	}
	res := make(map[uint]string, len(uws))
	for _, uw := range uws {
		res[uw.UserID] = uw.Role
	}
	return res, nil
}

func (r *teamsRepo) Update(ctx context.Context, t *Team) error {
	return r.db.WithContext(ctx).Save(t).Error
}

func (r *teamsRepo) Delete(ctx context.Context, t *Team) error {
	return r.db.WithContext(ctx).Delete(t).Error
}
