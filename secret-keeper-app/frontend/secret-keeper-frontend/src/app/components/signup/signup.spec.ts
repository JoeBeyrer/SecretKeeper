import { ComponentFixture, TestBed } from '@angular/core/testing';
import { ReactiveFormsModule } from '@angular/forms';
import { Router } from '@angular/router';
import { provideHttpClient } from '@angular/common/http';
import { provideHttpClientTesting } from '@angular/common/http/testing';
import { vi } from 'vitest';

import { Signup } from './signup';

describe('Signup', () => {
  let component: Signup;
  let fixture: ComponentFixture<Signup>;
  let routerSpy: { navigate: ReturnType<typeof vi.fn> };

  beforeEach(async () => {
    routerSpy = { navigate: vi.fn() };

    await TestBed.configureTestingModule({
      imports: [Signup, ReactiveFormsModule],
      providers: [
        provideHttpClient(),
        provideHttpClientTesting(),
        { provide: Router, useValue: routerSpy },
      ],
    }).compileComponents();

    fixture = TestBed.createComponent(Signup);
    component = fixture.componentInstance;
    fixture.detectChanges();
  });

  it('should create', () => {
    expect(component).toBeTruthy();
  });

  it('should initialize the form with empty fields', () => {
    expect(component.signupForm.value).toEqual({
      username: '',
      email: '',
      password: '',
      confirmPassword: '',
    });
  });

  it('should set errorMessage when username is missing', () => {
    component.signupForm.setValue({ username: '', email: 'a@b.com', password: 'password1', confirmPassword: 'password1' });
    component.onSubmit();
    expect(component.errorMessage).toBe('Username is required');
  });

  it('should set errorMessage when username is too short', () => {
    component.signupForm.setValue({ username: 'ab', email: 'a@b.com', password: 'password1', confirmPassword: 'password1' });
    component.onSubmit();
    expect(component.errorMessage).toBe('Username must be at least 3 characters');
  });

  it('should set errorMessage when email is missing', () => {
    component.signupForm.setValue({ username: 'alice', email: '', password: 'password1', confirmPassword: 'password1' });
    component.onSubmit();
    expect(component.errorMessage).toBe('Email is required');
  });

  it('should set errorMessage when email is invalid', () => {
    component.signupForm.setValue({ username: 'alice', email: 'notanemail', password: 'password1', confirmPassword: 'password1' });
    component.onSubmit();
    expect(component.errorMessage).toBe('Email must be a valid email address');
  });

  it('should set errorMessage when password is missing', () => {
    component.signupForm.setValue({ username: 'alice', email: 'a@b.com', password: '', confirmPassword: '' });
    component.onSubmit();
    expect(component.errorMessage).toBe('Password is required');
  });

  it('should set errorMessage when password is too short', () => {
    component.signupForm.setValue({ username: 'alice', email: 'a@b.com', password: 'short', confirmPassword: 'short' });
    component.onSubmit();
    expect(component.errorMessage).toBe('Password must be at least 8 characters');
  });

  it('should set errorMessage when passwords do not match', () => {
    component.signupForm.setValue({ username: 'alice', email: 'a@b.com', password: 'password1', confirmPassword: 'different1' });
    component.onSubmit();
    expect(component.errorMessage).toBe('Passwords do not match');
  });

  it('should navigate to /login when goToLogin is called', () => {
    component.goToLogin();
    expect(routerSpy.navigate).toHaveBeenCalledWith(['/login']);
  });
});
