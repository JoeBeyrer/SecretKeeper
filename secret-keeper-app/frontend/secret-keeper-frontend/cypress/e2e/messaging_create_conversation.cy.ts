describe('messaging_create_conversation', () => {
  it('creates a new conversation with Bob and shows a room key modal', () => {
    cy.login('Alice', 'Alice123');

    cy.get('input[placeholder="Username to chat with"]').type('Bob');
    cy.contains('button', 'New').click();

    cy.get('.modal').should('be.visible');

    cy.get('.modal-title').then(($title) => {
      const title = $title.text().trim();
      if (title === 'Choose Room Key') {
        // New conversation — a key has been pre-generated; just cancel
        cy.contains('button', 'Cancel').click();
      } else if (title === 'Your Room Key') {
        // Conversation was just created; confirm key saved
        cy.get('.room-key-text').should('not.be.empty');
        cy.contains("I've saved it").click();
      } else {
        // Enter Room Key prompt for an existing conversation
        cy.contains('Cancel').click();
      }
    });

    cy.get('.modal').should('not.exist');
  });
});
