describe('messaging_nav_to_profile', () => {
  it('navigates from messaging to profile via the sidebar nav', () => {
    cy.login('Alice', 'Alice123');
    cy.get('.nav-icon-btn').eq(1).click();
    cy.url().should('include', '/profile');
    cy.contains('Settings').should('be.visible');
  });
});
