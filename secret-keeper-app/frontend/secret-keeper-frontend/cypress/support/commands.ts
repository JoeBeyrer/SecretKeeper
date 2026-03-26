declare namespace Cypress {
  interface Chainable {
    login(username: string, password: string): Chainable<void>;
  }
}

Cypress.Commands.add('login', (username: string, password: string) => {
  cy.visit('/login');
  cy.get('#username').type(username);
  cy.get('#password').type(password);
  cy.get('.login-button').click();
  cy.url().should('include', '/messaging');
});
