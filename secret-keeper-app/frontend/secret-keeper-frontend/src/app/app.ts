import { Component, signal } from '@angular/core';
import { RouterOutlet } from '@angular/router';

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
}
