import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ReactiveFormsModule } from '@angular/forms';
import { Router, ActivatedRoute } from '@angular/router';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { of } from 'rxjs';
import { vi } from 'vitest';

import { Login } from './login';

describe('Login', () => {
  let component: Login;
  let fixture: ComponentFixture<Login>;
  let routerSpy: { navigate: ReturnType<typeof vi.fn> };

  beforeEach(async () => {
    routerSpy = { navigate: vi.fn() };

    await TestBed.configureTestingModule({
      imports: [Login, ReactiveFormsModule],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: Router, useValue: routerSpy },
        { provide: ActivatedRoute, useValue: { queryParams: of({}) } },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(Login);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should initialize form with empty username and password', () => {
    expect(component.loginForm.value).toEqual({ username: '', password: '' });
  });

  it('should set errorMessage when submitting invalid form', () => {
    component.loginForm.setValue({ username: '', password: '' });
    component.onSubmit();
    expect(component.errorMessage).toBe('Please fill in all fields with valid entries');
  });

  it('should mark form invalid when username is too short', () => {
    component.loginForm.setValue({ username: 'ab', password: 'password1' });
    expect(component.loginForm.valid).toBe(false);
  });

  it('should mark form invalid when password is too short', () => {
    component.loginForm.setValue({ username: 'alice', password: 'short' });
    expect(component.loginForm.valid).toBe(false);
  });

  it('should mark form valid with correct username and password', () => {
    component.loginForm.setValue({ username: 'alice', password: 'password1' });
    expect(component.loginForm.valid).toBe(true);
  });

  it('should navigate to /signup when goToSignup is called', () => {
    component.goToSignup();
    expect(routerSpy.navigate).toHaveBeenCalledWith(['/signup']);
  });

  it('should navigate to /reset-password when goToForgotPassword is called', () => {
    component.goToForgotPassword();
    expect(routerSpy.navigate).toHaveBeenCalledWith(['/reset-password']);
  });

  it('should set successMessage when verified query param is true', async () => {
    TestBed.resetTestingModule();
    routerSpy = { navigate: vi.fn() };
    await TestBed.configureTestingModule({
      imports: [Login, ReactiveFormsModule],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: Router, useValue: routerSpy },
        { provide: ActivatedRoute, useValue: { queryParams: of({ verified: 'true' }) } },
      ],
    }).compileComponents();

    const f = TestBed.createComponent(Login);
    const c = f.componentInstance;
    f.detectChanges();
    expect(c.successMessage).toBe('Email verified! You can now log in.');
  });
});
