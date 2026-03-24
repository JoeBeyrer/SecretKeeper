import { Routes } from '@angular/router';
import { Login } from './components/login/login';
import { Messaging } from './components/messaging/messaging';
import { Signup } from './components/signup/signup';
import { PasswordReset } from './components/password-reset/password-reset';
import { Profile } from './components/profile/profile';
import { Friends } from './components/friends/friends';

export const routes: Routes = [
  { path: '', redirectTo: '/login', pathMatch: 'full' },
  { path: 'login', component: Login },
  { path: 'messaging', component: Messaging },
  { path: 'signup', component: Signup },
  { path: 'reset-password', component: PasswordReset },
  { path: 'profile', component: Profile },
  { path: 'friends', component: Friends },
];
