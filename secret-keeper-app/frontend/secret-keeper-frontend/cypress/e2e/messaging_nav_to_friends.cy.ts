describe('messaging_nav_to_friends', () => {
  it('navigates from messaging to friends via the sidebar nav', () => {
    cy.login('Alice', 'Alice123');
    cy.get('.nav-icon-btn').eq(2).click();
    cy.url().should('include', '/friends');
    cy.contains('Friends').should('be.visible');
  });
});
