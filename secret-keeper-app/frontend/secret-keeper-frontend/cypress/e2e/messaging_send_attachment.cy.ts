describe('messaging_send_attachment', () => {
  it('queues an attachment before send and shows it in the chat after sending', () => {
    cy.login('Alice', 'Alice123');

    cy.get('input[placeholder="Username to chat with"]').clear().type('Bob');
    cy.contains('button', 'New').click();

    cy.get('.modal').should('be.visible');
    cy.get('.modal-title').then(($title) => {
      const title = $title.text().trim();
      if (title === 'Choose Room Key') {
        cy.contains('button', 'Create Conversation').click();
        cy.contains('.modal-title', 'Your Room Key').should('be.visible');
        cy.contains('button', "I've saved it").click();
      } else if (title === 'Your Room Key') {
        cy.contains('button', "I've saved it").click();
      }
    });

    cy.get('.hidden-attachment-input').selectFile('cypress/fixtures/sample-upload.txt', { force: true });
    cy.contains('.pending-attachment-chip', 'sample-upload.txt').should('be.visible');

    cy.get('.message-input-bar .rectangle-1-3').click();

    cy.get('.pending-attachments-bar').should('not.exist');
    cy.contains('.message-attachment-name', 'sample-upload.txt', { timeout: 10000 }).should('be.visible');
    cy.contains('.message-attachment-download', 'Download').should('be.visible');
  });
});
