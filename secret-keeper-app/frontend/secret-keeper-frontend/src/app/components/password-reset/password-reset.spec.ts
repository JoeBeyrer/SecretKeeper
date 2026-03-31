import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ReactiveFormsModule } from '@angular/forms';
import { Router, ActivatedRoute } from '@angular/router';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { HttpTestingController } from '@angular/common/http/testing';
import { of } from 'rxjs';
import { vi } from 'vitest';

import { PasswordReset } from './password-reset';

describe('PasswordReset', () => {
  let component: PasswordReset;
  let fixture: ComponentFixture<PasswordReset>;
  let httpMock: HttpTestingController;
  let routerSpy: { navigate: ReturnType<typeof vi.fn> };

  async function setup(queryParams: Record<string, string> = {}) {
    routerSpy = { navigate: vi.fn() };
    await TestBed.configureTestingModule({
      imports: [PasswordReset, ReactiveFormsModule],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: Router, useValue: routerSpy },
        { provide: ActivatedRoute, useValue: { queryParams: of(queryParams) } },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(PasswordReset);
    component = fixture.componentInstance;
    httpMock = TestBed.inject(HttpTestingController);
    fixture.detectChanges();
  }

  afterEach(() => {
    httpMock.verify();
  });

  describe('Request page (no token)', () => {
    beforeEach(async () => {
      await setup();
    });

    it('should create', () => {
      expect(component).toBeTruthy();
    });

    it('should start in request state', () => {
      expect(component.pageState).toBe('request');
    });

    it('should set requestError when email is invalid', () => {
      component.requestForm.setValue({ email: 'notanemail' });
      component.onRequestSubmit();
      expect(component.requestError).toBeTruthy();
      httpMock.expectNone('http://localhost:8080/api/password-reset/request');
    });

    it('should POST to password-reset/request and set requestMessage on success', () => {
      component.requestForm.setValue({ email: 'alice@example.com' });
      component.onRequestSubmit();
      const req = httpMock.expectOne('http://localhost:8080/api/password-reset/request');
      expect(req.request.method).toBe('POST');
      expect(req.request.body).toEqual({ email: 'alice@example.com' });
      req.flush({ message: 'Reset link sent!' });
      expect(component.requestMessage).toBe('Reset link sent!');
      expect(component.isLoading).toBe(false);
    });

    it('should set requestError on POST failure', () => {
      component.requestForm.setValue({ email: 'alice@example.com' });
      component.onRequestSubmit();
      const req = httpMock.expectOne('http://localhost:8080/api/password-reset/request');
      req.flush('error', { status: 500, statusText: 'Server Error' });
      expect(component.requestError).toBeTruthy();
      expect(component.isLoading).toBe(false);
    });

    it('goToLogin() should navigate to /login', () => {
      component.goToLogin();
      expect(routerSpy.navigate).toHaveBeenCalledWith(['/login']);
    });

    it('requestNewLink() should navigate to /reset-password', () => {
      component.requestNewLink();
      expect(routerSpy.navigate).toHaveBeenCalledWith(['/reset-password']);
    });
  });

  describe('Token validation', () => {
    it('should enter validating state and set reset state when token is valid', async () => {
      await setup({ token: 'valid-token-abc' });
      const req = httpMock.expectOne((r) => r.url.includes('validate'));
      req.flush({ valid: true });
      expect(component.pageState).toBe('reset');
      expect(component.token).toBe('valid-token-abc');
    });

    it('should set expired state when token is invalid', async () => {
      await setup({ token: 'bad-token' });
      const req = httpMock.expectOne((r) => r.url.includes('validate'));
      req.flush({ valid: false });
      expect(component.pageState).toBe('expired');
    });

    it('should set expired state when validation request errors', async () => {
      await setup({ token: 'bad-token' });
      const req = httpMock.expectOne((r) => r.url.includes('validate'));
      req.flush('error', { status: 500, statusText: 'Server Error' });
      expect(component.pageState).toBe('expired');
    });
  });

  describe('Reset page (valid token already set)', () => {
    beforeEach(async () => {
      await setup({ token: 'valid-token-abc' });
      const req = httpMock.expectOne((r) => r.url.includes('validate'));
      req.flush({ valid: true });
    });

    it('should show reset page after valid token', () => {
      expect(component.pageState).toBe('reset');
    });

    it('should set resetError when passwords do not match', () => {
      component.resetForm.setValue({ password: 'password1', confirmPassword: 'different1' });
      component.onResetSubmit();
      expect(component.resetError).toContain('do not match');
      httpMock.expectNone('http://localhost:8080/api/password-reset/confirm');
    });

    it('should set resetError when password is too short', () => {
      component.resetForm.setValue({ password: 'short', confirmPassword: 'short' });
      component.onResetSubmit();
      expect(component.resetError).toBeTruthy();
      httpMock.expectNone('http://localhost:8080/api/password-reset/confirm');
    });

    it('should POST confirm and set success state on success', () => {
      component.resetForm.setValue({ password: 'newpassword1', confirmPassword: 'newpassword1' });
      component.onResetSubmit();
      const req = httpMock.expectOne('http://localhost:8080/api/password-reset/confirm');
      expect(req.request.method).toBe('POST');
      expect(req.request.body).toEqual({ token: 'valid-token-abc', password: 'newpassword1' });
      req.flush({ message: 'Password reset!' });
      expect(component.pageState).toBe('success');
      expect(component.isLoading).toBe(false);
    });

    it('should set resetError with 422 expired link message', () => {
      component.resetForm.setValue({ password: 'newpassword1', confirmPassword: 'newpassword1' });
      component.onResetSubmit();
      const req = httpMock.expectOne('http://localhost:8080/api/password-reset/confirm');
      req.flush('expired', { status: 422, statusText: 'Unprocessable Entity' });
      expect(component.resetError).toContain('expired');
    });

    it('should set generic resetError on other confirm errors', () => {
      component.resetForm.setValue({ password: 'newpassword1', confirmPassword: 'newpassword1' });
      component.onResetSubmit();
      const req = httpMock.expectOne('http://localhost:8080/api/password-reset/confirm');
      req.flush('error', { status: 500, statusText: 'Server Error' });
      expect(component.resetError).toContain('Something went wrong');
    });
  });
});
