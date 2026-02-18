import { Component, OnInit } from '@angular/core';
import { FormBuilder, FormGroup, ReactiveFormsModule, Validators, AbstractControl, ValidationErrors } from '@angular/forms';
import { ActivatedRoute, Router } from '@angular/router';
import { HttpClient } from '@angular/common/http';

type PageState = 'request' | 'validating' | 'reset' | 'expired' | 'success';

function passwordMatchValidator(control: AbstractControl): ValidationErrors | null {
  const pw = control.get('password');
  const confirm = control.get('confirmPassword');
  if (pw && confirm && pw.value !== confirm.value) {
    return { passwordMismatch: true };
  }
  return null;
}

@Component({
  selector: 'app-password-reset',
  imports: [ReactiveFormsModule],
  templateUrl: './password-reset.html',
  styleUrl: './password-reset.css',
})
export class PasswordReset implements OnInit {
  pageState: PageState = 'request';
  token: string = '';

  requestForm: FormGroup;
  resetForm: FormGroup;

  requestMessage: string = '';
  requestError: string = '';
  resetError: string = '';
  isLoading: boolean = false;

  constructor(
    private http: HttpClient,
    private fb: FormBuilder,
    private route: ActivatedRoute,
    private router: Router
  ) {
    this.requestForm = this.fb.group({
      email: ['', [Validators.required, Validators.email]],
    });

    this.resetForm = this.fb.group(
      {
        password: ['', [Validators.required, Validators.minLength(8)]],
        confirmPassword: ['', [Validators.required]],
      },
      { validators: passwordMatchValidator }
    );
  }

  ngOnInit(): void {
    this.route.queryParams.subscribe((params) => {
      const token = params['token'];
      if (token) {
        this.token = token;
        this.pageState = 'validating';
        this.validateToken(token);
      }
    });
  }

  onRequestSubmit(): void {
    if (this.requestForm.invalid) {
      this.requestError = 'Please enter a valid email address.';
      return;
    }

    this.isLoading = true;
    this.requestError = '';
    this.requestMessage = '';

    const { email } = this.requestForm.value;

    this.http
      .post<{ message: string }>('http://localhost:8080/api/password-reset/request', { email })
      .subscribe({
        next: (res) => {
          this.requestMessage = res.message;
          this.isLoading = false;
        },
        error: () => {
          this.requestError = 'Something went wrong sending the email. Please try again.';
          this.isLoading = false;
        },
      });
  }

  private validateToken(token: string): void {
    this.http
      .get<{ valid: boolean }>(
        `http://localhost:8080/api/password-reset/validate?token=${encodeURIComponent(token)}`
      )
      .subscribe({
        next: (res) => {
          this.pageState = res.valid ? 'reset' : 'expired';
        },
        error: () => {
          this.pageState = 'expired';
        },
      });
  }

  onResetSubmit(): void {
    if (this.resetForm.errors?.['passwordMismatch']) {
      this.resetError = 'Passwords do not match.';
      return;
    }
    if (this.resetForm.invalid) {
      this.resetError = 'Password must be at least 8 characters.';
      return;
    }

    this.isLoading = true;
    this.resetError = '';

    const { password } = this.resetForm.value;

    this.http
      .post<{ message: string }>('http://localhost:8080/api/password-reset/confirm', {
        token: this.token,
        password,
      })
      .subscribe({
        next: () => {
          this.pageState = 'success';
          this.isLoading = false;
        },
        error: (err) => {
          this.resetError =
            err.status === 422
              ? 'This reset link has expired or already been used. Please request a new one.'
              : 'Something went wrong. Please try again.';
          this.isLoading = false;
        },
      });
  }

  goToLogin(): void {
    this.router.navigate(['/login']);
  }

  requestNewLink(): void {
    this.router.navigate(['/reset-password']);
  }
}
