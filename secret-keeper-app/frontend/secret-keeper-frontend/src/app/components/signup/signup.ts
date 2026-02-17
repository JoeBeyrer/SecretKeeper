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
    const {username,email,password,confirmPassword} = this.signupForm.value;
    if (this.signupForm.valid) {
      if(password != confirmPassword){
        this.errorMessage = 'Passwords do not match, please fix';
        return;
      }
      console.log('Information passed basic checks, sending request')
      this.http.post('http://localhost:8080/api/register', {username, email, password}).subscribe({
        next: () => {
          this.errorMessage = '';
          console.log('Successfully registered with username ', { username });
          this.router.navigate(['/login']);
        },
        error: (err) => {
          this.errorMessage = err;
          console.log('Register did not work with username ', { username });
          return;
        },
        complete: () => {
          this.errorMessage = '';
          console.log('Sign up request finished');
        }
      });
    } else {
      this.errorMessage = 'Please fill in all fields with valid entries';
      console.log('Invalid loginForm');
    }
  this.router.navigate(['/login']);
  }
}
