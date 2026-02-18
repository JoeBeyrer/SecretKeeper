import { Routes } from '@angular/router';
import { Login } from './components/login/login';
import { Messaging } from './components/messaging/messaging';
import { Signup } from './components/signup/signup';
import { PasswordReset } from './components/password-reset/password-reset';

export const routes: Routes = [
  { path: '', redirectTo: '/login', pathMatch: 'full' },
  { path: 'login', component: Login },
  { path: 'messaging', component: Messaging },
  { path: 'signup', component: Signup },
  { path: 'reset-password', component: PasswordReset }
];
