package service

import (
	"context"
	"sync"
	"time"

	"dodevops-api/api/monitor/dao"
	"dodevops-api/api/monitor/model"
	"dodevops-api/pkg/log"
)

const domainCertCheckInterval = 5 * time.Minute

// RunDomainCertChecks belongs to the API lifecycle, independent of open pages.
// Checks persist observations only; the existing alert rule engine sends alerts.
func RunDomainCertChecks(ctx context.Context) {
	s := &DomainCertServiceImpl{dao: dao.NewDomainCertDao()}
	runDomainCertChecks(ctx, domainCertCheckInterval, s.dao.GetAll, func(ctx context.Context, cert *model.DomainCert) {
		s.checkCertContext(ctx, cert)
		// Shutdown is not a failed certificate check.
		if ctx.Err() != nil {
			return
		}
		if err := s.dao.UpdateCheck(cert); err != nil {
			log.Log().Errorf("domain certificate %d: persist check: %v", cert.ID, err)
		}
	})
}

func runDomainCertChecks(ctx context.Context, interval time.Duration, list func() ([]model.DomainCert, error), check func(context.Context, *model.DomainCert)) {
	for ctx.Err() == nil {
		certs, err := list()
		if err != nil {
			log.Log().Errorf("domain certificate collector: %v", err)
		} else {
			jobs := make(chan *model.DomainCert)
			var workers sync.WaitGroup
			for i := 0; i < 4; i++ {
				workers.Add(1)
				go func() {
					defer workers.Done()
					for cert := range jobs {
						if ctx.Err() != nil {
							return
						}
						check(ctx, cert)
					}
				}()
			}
		dispatch:
			for i := range certs {
				select {
				case jobs <- &certs[i]:
				case <-ctx.Done():
					break dispatch
				}
			}
			close(jobs)
			workers.Wait()
		}
		// Wait after completion: slow endpoints never cause overlapping sweeps.
		timer := time.NewTimer(interval)
		select {
		case <-ctx.Done():
			timer.Stop()
			return
		case <-timer.C:
		}
	}
}
