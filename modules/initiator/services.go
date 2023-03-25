// Copyright 2022 The Gitea Authors. All rights reserved.
// SPDX-License-Identifier: MIT

package initiator

import (
	"context"
	"fmt"

	"code.gitea.io/gitea/modules/log"
)

type ServiceConfig struct {
	Name         string
	Init         func(ctx context.Context) error
	Shutdown     func(ctx context.Context) error
	Dependencies []string
}

type service struct {
	initialized bool
	cfg         *ServiceConfig
	dependents  []string
}

var services = make(map[string]*service)

func RegisterService(serviceCfg *ServiceConfig) {
	if serviceCfg.Name == "" {
		log.Fatal("Service configuration %#v has no name", serviceCfg)
	}
	if _, ok := services[serviceCfg.Name]; ok {
		log.Fatal("A service named %s exist", serviceCfg.Name)
	}

	services[serviceCfg.Name] = &service{
		cfg: serviceCfg,
	}
}

func initService(ctx context.Context, service *service) error {
	if service.initialized {
		return nil
	}
	for _, dep := range service.cfg.Dependencies {
		depService, ok := services[dep]
		if !ok {
			return fmt.Errorf("service %s depends on %s which does not exist", service.cfg.Name, dep)
		}
		if err := initService(ctx, depService); err != nil {
			return err
		}
		depService.dependents = append(depService.dependents, service.cfg.Name)
	}
	log.Trace("Initializing service: %s", service.cfg.Name)
	if service.cfg.Init == nil {
		return fmt.Errorf("service %s has no init function", service.cfg.Name)
	}
	if err := service.cfg.Init(ctx); err != nil {
		return err
	}
	service.initialized = true
	return nil
}

func Init(ctx context.Context) error {
	for _, service := range services {
		if err := initService(ctx, service); err != nil {
			return err
		}
	}
	return nil
}

func shutdownService(ctx context.Context, service *service) error {
	if !service.initialized {
		return nil
	}

	for _, dep := range service.dependents {
		if err := shutdownService(ctx, services[dep]); err != nil {
			return err
		}
	}

	log.Trace("Shuting down service: %s", service.cfg.Name)
	if service.cfg.Shutdown == nil {
		service.initialized = false
		return nil
	}

	if err := service.cfg.Shutdown(ctx); err != nil {
		return err
	}
	service.initialized = false
	return nil
}

func ShutdownService(ctx context.Context) error {
	for _, service := range services {
		if err := shutdownService(ctx, service); err != nil {
			return err
		}
	}
	return nil
}
