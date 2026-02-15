import { Component } from '@angular/core';
import { FormBuilder, FormGroup, ReactiveFormsModule } from '@angular/forms';
import { Router } from '@angular/router';

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
    private fb: FormBuilder,
    private router: Router
  ) {
    this.signupForm = this.fb.group({
      username: [''],
      email: [''],
      password: [''],
      confirmPassword: ['']
    });
  }

  onSubmit() {
    // have to add logic here, for right now the command
    // below just redirects to login
    this.router.navigate(['/login']);
  }
}
