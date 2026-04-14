describe('Signup password validation', () => {
  beforeEach(() => {
    cy.visit('/signup');
  });

  it('rejects a password that is too short', () => {
    cy.get('#username').type('Alin_Dobra');
    cy.get('#email').type('alindobra@secretkeeper.com');
    cy.get('#password').type('pass');
    cy.get('#confirmPassword').type('pass');
    cy.get('.signup-button').click();
    cy.get('.feedback.error').should('contain', 'Password must be at least 8 characters');
  });
});
