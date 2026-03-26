describe('messaging_send_message', () => {
  it('opens a conversation with Bob and handles the room key modal', () => {
    cy.login('Alice', 'Alice123');

    cy.get('input[placeholder="Username to chat with"]').type('Bob');
    cy.contains('button', 'New').click();

    cy.get('.modal').should('be.visible');

    cy.get('.modal').then(($modal) => {
      if ($modal.text().includes('Your Room Key')) {
        cy.get('.room-key-text').should('not.be.empty');
        cy.contains("I've saved it").click();
        cy.get('.modal').should('not.exist');
        cy.contains('End-to-end encrypted').should('be.visible');
        cy.get('input[placeholder="Type your message here..."]').type('Hello Bob from Cypress!');
        cy.get('.message-input-bar .rectangle-1-3').click();
        cy.get('.message-row.mine').should('exist');
        cy.contains('Hello Bob from Cypress!').should('be.visible');
        cy.get('input[placeholder="Type your message here..."]').should('have.value', '');
      } else {
        cy.contains('Enter Room Key').should('be.visible');
        cy.contains('Cancel').click();
        cy.get('.modal').should('not.exist');
      }
    });
  });
});
