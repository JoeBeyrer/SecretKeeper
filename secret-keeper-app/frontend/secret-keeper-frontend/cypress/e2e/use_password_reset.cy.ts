describe('use_password_reset', () => {
  it('submits email and shows success message', () => {
    cy.intercept('POST', '**/api/password-reset/request', {
      statusCode: 200,
      body: { message: 'If that email is registered, a reset link has been sent.' },
    }).as('requestReset');

    cy.visit('/reset-password');
    cy.get('input[formControlName="email"]').type('alindobra@secretkeeper.com');
    cy.get('.reset-button').click();

    cy.wait('@requestReset');
    cy.get('.success-message').should('contain', 'reset link has been sent');
  });

  it('shows error for invalid email', () => {
    cy.visit('/reset-password');
    cy.get('input[formControlName="email"]').type('notanemail');
    cy.get('.reset-button').click();
    cy.get('.error-message').should('contain', 'valid email');
  });
});
