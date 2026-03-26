describe('messaging_load_page', () => {
  it('loads the messaging page after login and shows the sidebar', () => {
    cy.login('Alice', 'Alice123');
    cy.url().should('include', '/messaging');
    cy.contains('Chats').should('be.visible');
    cy.contains('Select a conversation').should('be.visible');
  });
});
