import { Injectable } from '@angular/core';

@Injectable({ providedIn: 'root' })
export class ThemeService {
  private _isDark = true;

  get isDark(): boolean {
    return this._isDark;
  }

  constructor() {
    const saved = localStorage.getItem('sk-theme');
    this._isDark = saved !== 'light';
    this.apply();
  }

  toggle(): void {
    this._isDark = !this._isDark;
    localStorage.setItem('sk-theme', this._isDark ? 'dark' : 'light');
    this.apply();
  }

  private apply(): void {
    document.body.classList.toggle('light-mode', !this._isDark);
  }
}
