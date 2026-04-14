describe('friends_load_page', () => {
  it('loads the friends page and shows tabs', () => {
    cy.login('Alice', 'Alice123');
    cy.visit('/friends');

    cy.contains('Friends').should('be.visible');
    cy.get('.tab').should('have.length', 4);
    cy.contains('Requests').should('be.visible');
    cy.contains('Add Friend').should('be.visible');
    cy.contains('Search Users').should('be.visible');
  });
});
