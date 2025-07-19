package hooks

import (
	"github.com/pocketbase/pocketbase/core"
	"github.com/tigawanna/pockestrator/internal/services"
)

// ServiceHooks handles PocketBase event hooks for service lifecycle
type ServiceHooks struct {
	pbManager  services.PocketBaseManager
	systemdMgr services.SystemdManager
	caddyMgr   services.CaddyManager
	validator  services.ValidationService
}

// NewServiceHooks creates a new ServiceHooks instance
func NewServiceHooks(
	pbManager services.PocketBaseManager,
	systemdMgr services.SystemdManager,
	caddyMgr services.CaddyManager,
	validator services.ValidationService,
) *ServiceHooks {
	return &ServiceHooks{
		pbManager:  pbManager,
		systemdMgr: systemdMgr,
		caddyMgr:   caddyMgr,
		validator:  validator,
	}
}

// OnServiceCreate handles service creation events
func (h *ServiceHooks) OnServiceCreate(e *core.RecordEvent) error {
	// TODO: Implement service creation hook
	return nil
}

// OnServiceUpdate handles service update events
func (h *ServiceHooks) OnServiceUpdate(e *core.RecordEvent) error {
	// TODO: Implement service update hook
	return nil
}

// OnServiceDelete handles service deletion events
func (h *ServiceHooks) OnServiceDelete(e *core.RecordEvent) error {
	// TODO: Implement service deletion hook
	return nil
}
