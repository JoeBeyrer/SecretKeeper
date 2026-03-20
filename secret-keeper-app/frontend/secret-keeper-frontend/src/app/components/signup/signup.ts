import { Component } from '@angular/core';
import { FormBuilder, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { Router } from '@angular/router';
import { HttpClient } from '@angular/common/http';

@Component({
  selector: 'app-signup',
  imports: [ReactiveFormsModule],
  templateUrl: './signup.html',
  styleUrl: './signup.css',
})
export class Signup {
  signupForm: FormGroup;
  errorMessage: string = '';
  successMessage: string = '';

  constructor(
    private http: HttpClient,
    private fb: FormBuilder,
    private router: Router
  ) {
    this.signupForm = this.fb.group({
      username: ['', [Validators.required, Validators.minLength(3)]],
      email: ['', [Validators.required, Validators.email]],
      password: ['', [Validators.required, Validators.minLength(8)]],
      confirmPassword: ['', [Validators.required]]
    });
  }

  onSubmit() {
    const { username, email, password, confirmPassword } = this.signupForm.value;
    const controls = this.signupForm.controls;

    if (!username) {
      this.errorMessage = 'Username is required';
      return;
    }
    if (controls['username'].errors?.['minlength']) {
      this.errorMessage = 'Username must be at least 3 characters';
      return;
    }
    if (!email) {
      this.errorMessage = 'Email is required';
      return;
    }
    if (controls['email'].errors?.['email']) {
      this.errorMessage = 'Email must be a valid email address';
      return;
    }
    if (!password) {
      this.errorMessage = 'Password is required';
      return;
    }
    if (controls['password'].errors?.['minlength']) {
      this.errorMessage = 'Password must be at least 8 characters';
      return;
    }
    if (!confirmPassword) {
      this.errorMessage = 'Please confirm your password';
      return;
    }
    if (password !== confirmPassword) {
      this.errorMessage = 'Passwords do not match';
      return;
    }

    this.errorMessage = '';
    console.log('Information passed basic checks, sending request');
    this.http.post<{ message: string }>('http://localhost:8080/api/register', { username, email, password }).subscribe({
      next: (res) => {
        this.errorMessage = '';
        this.successMessage = res.message || 'Account created! Please check your email to verify your address before logging in.';
        this.signupForm.reset();
        console.log('Successfully registered with username', username);
      },
      error: (err) => {
        this.successMessage = '';
        if (err.status === 409) {
          this.errorMessage = 'Username or email is already taken.';
        } else {
          this.errorMessage = 'Something went wrong. Please try again.';
        }
        console.log('Register did not work:', err);
      },
    });
  }

  goToLogin() {
    this.router.navigate(['/login']);
  }
}
