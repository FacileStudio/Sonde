package schemas

import "gorm.io/gorm"

func Migrate(db *gorm.DB) error {
	if err := db.AutoMigrate(&User{}, &Session{}, &Event{}, &Ticket{}); err != nil {
		return err
	}

	return db.Exec(`
		DO $$
		BEGIN
			ALTER TABLE tickets
			ADD CONSTRAINT tickets_status_check CHECK (status IN ('valid', 'used', 'revoked'));
		EXCEPTION
			WHEN duplicate_object THEN NULL;
		END $$;
	`).Error
}
