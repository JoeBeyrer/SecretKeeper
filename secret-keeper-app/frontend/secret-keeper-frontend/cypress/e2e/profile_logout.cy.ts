describe('profile_logout', () => {
  it('logs the user out and redirects to login', () => {
    cy.login('Alice', 'Alice123');
    cy.visit('/profile');

    cy.get('.btn-logout').click();
    cy.url().should('include', '/login');
  });
});
