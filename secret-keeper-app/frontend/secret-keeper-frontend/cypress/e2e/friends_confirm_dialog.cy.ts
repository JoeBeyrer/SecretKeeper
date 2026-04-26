describe('friends_confirm_dialog', () => {
  beforeEach(() => {
    cy.login('Alice', 'Alice123');
    cy.visit('/friends');
  });

  it('shows a confirmation dialog when clicking Remove on a friend', () => {
    cy.get('.friends-panel .panel-body').then($body => {
      if ($body.find('.action-btn.danger').length > 0) {
        cy.get('.action-btn.danger').first().click();
        cy.get('.confirm-overlay').should('be.visible');
        cy.get('.confirm-msg').should('be.visible');
        cy.get('.confirm-cancel').should('be.visible');
        cy.get('.confirm-ok').should('be.visible');
      } else {
        cy.log('No friends to remove — skipping dialog assertion');
      }
    });
  });

  it('dismisses the dialog when Cancel is clicked', () => {
    cy.get('.friends-panel .panel-body').then($body => {
      if ($body.find('.action-btn.danger').length > 0) {
        cy.get('.action-btn.danger').first().click();
        cy.get('.confirm-overlay').should('be.visible');
        cy.get('.confirm-cancel').click();
        cy.get('.confirm-overlay').should('not.exist');
      } else {
        cy.log('No friends to remove — skipping dialog assertion');
      }
    });
  });

  it('dismisses the dialog when clicking the overlay backdrop', () => {
    cy.get('.friends-panel .panel-body').then($body => {
      if ($body.find('.action-btn.danger').length > 0) {
        cy.get('.action-btn.danger').first().click();
        cy.get('.confirm-overlay').should('be.visible');
        cy.get('.confirm-overlay').click({ force: true });
        cy.get('.confirm-overlay').should('not.exist');
      } else {
        cy.log('No friends to remove — skipping dialog assertion');
      }
    });
  });
});
