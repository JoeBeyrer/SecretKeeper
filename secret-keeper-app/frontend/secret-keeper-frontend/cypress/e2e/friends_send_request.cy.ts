describe('friends_send_request', () => {
  it('sends a friend request from the Add Friend tab', () => {
    cy.login('Alice', 'Alice123');
    cy.visit('/friends');

    cy.contains('.tab', 'Add Friend').click();

    cy.get('.add-section .field-input').type('Rob');
    cy.get('.send-btn').click();

    cy.get('.feedback').should('be.visible');
  });
});
