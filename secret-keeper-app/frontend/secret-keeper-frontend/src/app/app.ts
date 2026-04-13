import { Component, signal } from '@angular/core';
import { RouterOutlet } from '@angular/router';
import { ThemeService } from './services/theme.service';

@Component({
  selector: 'app-root',
  imports: [RouterOutlet],
  template: `
    <div class="app">
      <router-outlet></router-outlet>
    </div>
  `,
  styleUrl: './app.css'
})
export class App {
  protected readonly title = signal('secret-keeper-frontend');

  // Injecting ThemeService here ensures it initializes (reads localStorage + applies class)
  // before any routed component renders — preventing a flash of wrong theme.
  constructor(private _theme: ThemeService) {}
}
