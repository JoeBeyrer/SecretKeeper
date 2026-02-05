import { Routes } from '@angular/router';
import { Login } from './components/login/login';
import { Messaging } from './components/messaging/messaging';

export const routes: Routes = [
  { path: '', redirectTo: '/login', pathMatch: 'full' },
  { path: 'login', component: Login },
  { path: 'messaging', component: Messaging }
];