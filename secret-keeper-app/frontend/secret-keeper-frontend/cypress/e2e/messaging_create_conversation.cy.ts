describe('messaging_create_conversation', () => {
  it('creates a new conversation with Bob and shows a room key modal', () => {
    cy.login('Alice', 'Alice123');

    cy.get('input[placeholder="Username to chat with"]').type('Bob');
    cy.contains('button', 'New').click();

    cy.get('.modal').should('be.visible');

    cy.get('.modal').then(($modal) => {
      if ($modal.text().includes('Your Room Key')) {
        cy.get('.room-key-text').should('not.be.empty');
        cy.contains("I've saved it").click();
      } else {
        cy.contains('Enter Room Key').should('be.visible');
        cy.contains('Cancel').click();
      }
    });

    cy.get('.modal').should('not.exist');
  });
});
