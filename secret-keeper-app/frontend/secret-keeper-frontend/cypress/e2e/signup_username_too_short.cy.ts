describe('Signup username validation', () => {
  beforeEach(() => {
    cy.visit('/signup');
  });

  it('rejects a username that is too short', () => {
    cy.get('#username').type('Al');
    cy.get('#email').type('alindobra@secretkeeper.com');
    cy.get('#password').type('password123');
    cy.get('#confirmPassword').type('password123');
    cy.get('.signup-button').click();
    cy.get('.error-message').should('contain', 'Username must be at least 3 characters');
  });
});
