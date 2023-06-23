import { TestBed } from '@angular/core/testing';

import { MonitoringServiceService } from './monitoring-service.service';

describe('MonitoringServiceService', () => {
  let service: MonitoringServiceService;

  beforeEach(() => {
    TestBed.configureTestingModule({});
    service = TestBed.inject(MonitoringServiceService);
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });
});
