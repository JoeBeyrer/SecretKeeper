// clearDB ensures no conversation exists so chatWith=Bob always shows
// "Choose Room Key" making the modal state deterministic.
describe('messaging_create_conversation', { testIsolation: false }, () => {
  before(() => {
    cy.task('clearDB');
    cy.login('Alice', 'Alice123');
  });

  it('opens the create conversation modal and cancels', () => {
    cy.contains('button', 'Create Conversation').click();
    cy.get('.modal').should('be.visible');
    cy.get('.modal-title').should('contain', 'Create Conversation');
    cy.contains('button', 'Cancel').click();
    cy.get('.modal').should('not.exist');
  });

  it('shows Choose Room Key modal for Bob via chatWith and cancels', () => {
    cy.visit('/messaging?chatWith=Bob');
    cy.get('.modal').should('be.visible');
    cy.get('.modal-title').should('contain', 'Choose Room Key');
    cy.contains('button', 'Cancel').click();
    cy.get('.modal').should('not.exist');
  });
});
