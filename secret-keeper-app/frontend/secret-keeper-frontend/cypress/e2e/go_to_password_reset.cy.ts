describe('go_to_password_reset', () => {
  it('navigates from login page to password reset page', () => {
    cy.visit('/login');
    cy.contains('Reset it here').click();
    cy.url().should('include', '/reset-password');
  });
});
