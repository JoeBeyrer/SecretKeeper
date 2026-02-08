import { Component, OnInit } from '@angular/core';
import { FormBuilder, FormGroup, ReactiveFormsModule, Validators } from '@angular/forms';
import { Router } from '@angular/router';
import { HttpClient } from '@angular/common/http';

@Component({
  selector: 'app-login',
  imports: [ReactiveFormsModule],
  templateUrl: './login.html',
  styleUrl: './login.css',
})

export class Login implements OnInit {
  loginForm: FormGroup;
  errorMessage: string = '';
  validCredentials: [string, string][] = [];

  constructor(
    private fb: FormBuilder,
    private router: Router,
    private http: HttpClient
  ) {
    this.loginForm = this.fb.group({
      username: ['', [Validators.required]],
      password: ['', [Validators.required]]
    });
  }

  ngOnInit() {
    this.http.get('app/components/login/login.txt', { responseType: 'text' }) //getting info from login.txt
      .subscribe({
        next: (data) => {
          try {
            this.validCredentials = JSON.parse(data);
            console.log('got the credentials', this.validCredentials);
          } catch (error) {
            console.error('something went wrong parsing login.txt', error);
          }
        },
        error: (error) => {
          console.error('something went wrong loading login.txt', error);
        }
      });
  }

  onSubmit() {
    if (this.loginForm.valid) {
      const { username, password } = this.loginForm.value;
      
      const isValid = this.validCredentials.some(
        ([validUser, validPass]) => validUser === username && validPass === password
      );
      
      if (isValid) {
        this.errorMessage = '';
        console.log('login worked', { username });
        this.router.navigate(['/messaging']);
      } else {
        this.errorMessage = 'Login incorrect';
        console.log("login didn't work", { username });
      }
    } else {
      this.errorMessage = 'Fill in all fields';
    }
  }
}
