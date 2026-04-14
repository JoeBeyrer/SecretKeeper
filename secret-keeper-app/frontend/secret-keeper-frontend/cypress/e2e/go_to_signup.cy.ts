describe('go_to_signup', () => {
  it('navigates from login page to signup page', () => {
    cy.visit('/login');
    cy.contains('Sign up here').click();
    cy.url().should('include', '/signup');
  });
});
