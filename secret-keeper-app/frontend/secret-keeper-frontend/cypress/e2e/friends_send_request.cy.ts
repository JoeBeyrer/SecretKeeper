describe('friends_send_request', () => {
  it('sends a friend request to Bob from Alice', () => {
    cy.login('Alice', 'Alice123');
    cy.visit('/friends');

    cy.contains('.tab', 'Add Friend').click();

    cy.get('.add-section .form-control').type('Bob');
    cy.get('.send-btn').click();

    cy.get('.feedback').should('be.visible');
  });
});
