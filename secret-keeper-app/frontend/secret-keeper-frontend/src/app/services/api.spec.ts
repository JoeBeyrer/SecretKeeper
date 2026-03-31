import { TestBed } from '@angular/core/testing';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { HttpTestingController } from '@angular/common/http/testing';

import { ApiService } from './api';

describe('ApiService', () => {
  let service: ApiService;
  let httpMock: HttpTestingController;
  const base = 'http://localhost:8080/api';

  beforeEach(() => {
    TestBed.configureTestingModule({
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
      ],
    });
    service = TestBed.inject(ApiService);
    httpMock = TestBed.inject(HttpTestingController);
  });

  afterEach(() => {
    httpMock.verify();
  });

  it('should be created', () => {
    expect(service).toBeTruthy();
  });

  it('get() should send a GET request to the correct URL', () => {
    service.get('friends').subscribe();
    const req = httpMock.expectOne(`${base}/friends`);
    expect(req.request.method).toBe('GET');
    req.flush([]);
  });

  it('post() should send a POST request with the given body', () => {
    const body = { username: 'alice', password: 'pw123456' };
    service.post('login', body).subscribe();
    const req = httpMock.expectOne(`${base}/login`);
    expect(req.request.method).toBe('POST');
    expect(req.request.body).toEqual(body);
    req.flush({});
  });

  it('put() should send a PUT request with the given body', () => {
    const body = { display_name: 'Alice' };
    service.put('profile/update', body).subscribe();
    const req = httpMock.expectOne(`${base}/profile/update`);
    expect(req.request.method).toBe('PUT');
    expect(req.request.body).toEqual(body);
    req.flush({});
  });

  it('delete() should send a DELETE request to the correct URL', () => {
    service.delete('friends/remove').subscribe();
    const req = httpMock.expectOne(`${base}/friends/remove`);
    expect(req.request.method).toBe('DELETE');
    req.flush({});
  });
});
