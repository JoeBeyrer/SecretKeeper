describe('Signup success flow', () => {
  beforeEach(() => {
    cy.visit('/signup');
  });

  it('accepts valid credentials and shows success', () => {
    cy.intercept('POST', '**/api/register', {
      statusCode: 200,
      body: { message: 'Account created! Please check your email to verify your address before logging in.' },
    }).as('register');

    cy.get('#username').type('Alin_Dobra');
    cy.get('#email').type('alindobra@secretkeeper.com');
    cy.get('#password').type('SecurePass123');
    cy.get('#confirmPassword').type('SecurePass123');
    cy.get('.signup-button').click();

    cy.wait('@register');
    cy.get('.feedback.success').should('contain', 'Account created');
  });
});
