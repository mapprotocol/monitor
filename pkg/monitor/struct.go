package monitor

import "time"

type BridgeTransactionInfo struct {
	Id              int         `gorm:"column:id" db:"id" json:"id" form:"id"`
	SourceChainId   interface{} `gorm:"column:source_chain_id" db:"source_chain_id" json:"source_chain_id" form:"source_chain_id"`
	SourceHash      interface{} `gorm:"column:source_hash" db:"source_hash" json:"source_hash" form:"source_hash"`
	DestinationHash interface{} `gorm:"column:destination_hash" db:"destination_hash" json:"destination_hash" form:"destination_hash"`
	CompleteTime    *time.Time  `gorm:"column:complete_time" db:"complete_time" json:"complete_time" form:"complete_time"`
	Timestamp       *time.Time  `gorm:"column:timestamp" db:"timestamp" json:"timestamp" form:"timestamp"`
}
